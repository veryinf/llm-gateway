package handlers

import (
	"llm-gateway/internal/model"
	"llm-gateway/internal/web/common"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
)

type UserHandler struct {
	common.BaseHandler
}

type userWithCount struct {
	model.User
	UserKeyCount int `json:"userKeyCount"`
}

// SearchUsers 搜索用户
func (h *UserHandler) SearchUsers(c echo.Context) error {
	input := &common.SearchParams{}
	if err := c.Bind(input); err != nil {
		return err
	}
	query := h.DB.Model(&model.User{}).Order("uid DESC")
	// 关键词搜索：username、name、phone、department
	if pattern := input.EscapedKw(); pattern != "" {
		query = query.Where("username LIKE ? ESCAPE '\\' OR name LIKE ? ESCAPE '\\' OR phone LIKE ? ESCAPE '\\' OR department LIKE ? ESCAPE '\\'", pattern, pattern, pattern, pattern)
	}
	// 过滤条件
	for _, filter := range input.Filters {
		switch filter.Field {
		case "role":
			if err := validation.Validate(filter.Value, validation.Required, validation.In(string(model.RoleAdmin), string(model.RoleUser), string(model.RoleViewer))); err != nil {
				return h.Error(-11, err.Error())
			}
			query = query.Where("role = ?", filter.Value)
		case "status":
			if err := validation.Validate(filter.Value, validation.Required, validation.In("active", "disabled")); err != nil {
				return h.Error(-11, err.Error())
			}
			query = query.Where("status = ?", filter.Value)
		case "department":
			query = query.Where("department = ?", filter.Value)
		}
	}

	// 查询关联的 API Key 数量
	var count int64
	var users []model.User
	if err := h.Pagination(&input.Pagination, query, &users, &count); err != nil {
		return err
	}

	var counts []struct {
		UID   uint
		Count int
	}
	h.DB.Model(&model.UserKey{}).Select("uid, count(*) as count").Group("uid").Scan(&counts)
	countMap := make(map[uint]int, len(counts))
	for _, c := range counts {
		countMap[c.UID] = c.Count
	}

	result := make([]userWithCount, len(users))
	for i, u := range users {
		result[i] = userWithCount{User: u, UserKeyCount: countMap[u.UID]}
	}

	return common.NewDataSet(result, count)
}

// FetchUser 获取单个用户
func (h *UserHandler) FetchUser(c echo.Context) error {
	input := &struct {
		UID int64 `json:"uid"`
	}{}
	if err := c.Bind(input); err != nil {
		return err
	}
	if err := validation.ValidateStruct(input,
		validation.Field(&input.UID, validation.Required),
	); err != nil {
		return h.Error(-11, err.Error())
	}
	var user model.User
	if err := h.DB.First(&user, input.UID).Error; err != nil {
		return h.Error(-24, "用户不存在")
	}
	var userKeyCount int64
	h.DB.Model(&model.UserKey{}).Where("uid = ?", user.UID).Count(&userKeyCount)
	return common.NewData(userWithCount{User: user, UserKeyCount: int(userKeyCount)})
}

// AddUser 添加用户
func (h *UserHandler) AddUser(c echo.Context) error {
	input := &model.User{}
	if err := c.Bind(input); err != nil {
		return err
	}
	if err := validation.ValidateStruct(input,
		validation.Field(&input.Username, validation.Required),
		validation.Field(&input.Password, validation.Required),
		validation.Field(&input.Role, validation.Required, validation.In(string(model.RoleAdmin), string(model.RoleUser), string(model.RoleViewer))),
	); err != nil {
		return h.Error(-11, err.Error())
	}
	// 检查用户名唯一性
	var exist int64
	h.DB.Model(&model.User{}).Where("username = ?", input.Username).Count(&exist)
	if exist > 0 {
		return h.Error(-12, "用户名已存在")
	}
	// 密码加密
	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return h.Error(-21, "密码加密失败")
	}
	input.Password = string(hash)
	input.Status = "active"
	if err := h.DB.Create(input).Error; err != nil {
		return h.Error(-21, err.Error())
	}
	return common.NewData(input)
}

// UpdateUser 更新用户
func (h *UserHandler) UpdateUser(c echo.Context) error {
	input, err := h.GetJSON(c)
	if err != nil {
		return err
	}
	uid := input.Get("uid")
	if !uid.Exists() || uid.Uint() == 0 {
		return h.Error(-23, "uid is required")
	}
	newState := map[string]any{}
	if input.Get("username").Exists() {
		newState["username"] = input.Get("username").String()
	}
	if input.Get("password").Exists() && input.Get("password").String() != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(input.Get("password").String()), bcrypt.DefaultCost)
		if err != nil {
			return h.Error(-21, "密码加密失败")
		}
		newState["password"] = string(hash)
	}
	if input.Get("name").Exists() {
		newState["name"] = input.Get("name").String()
	}
	if input.Get("phone").Exists() {
		newState["phone"] = input.Get("phone").String()
	}
	if input.Get("department").Exists() {
		newState["department"] = input.Get("department").String()
	}
	if input.Get("role").Exists() {
		role := input.Get("role").String()
		if err := validation.Validate(role, validation.In(string(model.RoleAdmin), string(model.RoleUser), string(model.RoleViewer))); err != nil {
			return h.Error(-11, err.Error())
		}
		newState["role"] = role
	}
	if input.Get("status").Exists() {
		status := input.Get("status").String()
		if err := validation.Validate(status, validation.In("active", "disabled")); err != nil {
			return h.Error(-11, err.Error())
		}
		newState["status"] = status
	}
	if len(newState) == 0 {
		return h.Success()
	}
	if err := h.DB.Model(&model.User{}).Where("uid = ?", uid.Uint()).Updates(newState).Error; err != nil {
		return h.Error(-22, err.Error())
	}
	return h.Success()
}

// RemoveUser 删除用户
func (h *UserHandler) RemoveUser(c echo.Context) error {
	input := &struct {
		UID int64 `json:"uid"`
	}{}
	if err := c.Bind(input); err != nil {
		return err
	}
	if err := validation.ValidateStruct(input,
		validation.Field(&input.UID, validation.Required),
	); err != nil {
		return h.Error(-11, err.Error())
	}
	if err := h.DB.Delete(&model.User{}, input.UID).Error; err != nil {
		return h.Error(-23, err.Error())
	}
	return h.Success()
}

func (h *UserHandler) RegisterRoutes(g *echo.Group) {
	g.POST("/user/search", h.SearchUsers)
	g.POST("/user/fetch", h.FetchUser)
	g.POST("/user/add", h.AddUser)
	g.POST("/user/update", h.UpdateUser)
	g.POST("/user/remove", h.RemoveUser)
}
