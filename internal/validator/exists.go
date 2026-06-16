package validator

import (
	"fmt"
	"llm-gateway/internal/core"
	"log/slog"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/samber/lo"
	"gorm.io/gorm"
)

type ExistsRule struct {
	model any
	field string
	err   error
}

func Exists(model any) ExistsRule {
	migrator := core.DB.Model(model).Migrator()
	if !migrator.HasTable(model) {
		slog.Error("验证器配置错误: 表不存在", "model", fmt.Sprintf("%T", model))
		return ExistsRule{model: model, field: "???", err: fmt.Errorf("表不存在: %T", model)}
	}
	columns, _ := migrator.ColumnTypes(model)
	primaryKeys := lo.Filter(columns, func(c gorm.ColumnType, i int) bool { p, _ := c.PrimaryKey(); return p })
	if len(primaryKeys) != 1 {
		slog.Error("验证器配置错误: 仅支持单主键", "model", fmt.Sprintf("%T", model))
		return ExistsRule{model: model, field: "???", err: fmt.Errorf("仅支持单主键的表: %T", model)}
	}
	return ExistsByField(model, primaryKeys[0].Name())
}

func ExistsByField(model any, field string) ExistsRule {
	return ExistsRule{
		model: model,
		field: field,
	}
}

func (e ExistsRule) Validate(value any) error {
	if e.err != nil {
		return validation.NewError("validation_config", e.err.Error())
	}
	value, isNil := validation.Indirect(value)
	if isNil || validation.IsEmpty(value) {
		return nil
	}

	var count int64
	err := core.DB.Model(e.model).Where(e.field+"=?", value).Limit(1).Count(&count).Error
	if count > 0 {
		return nil
	}

	if err != nil {
		return validation.NewError("validation_exists", "判断关联字段出错, 请检查详情: "+err.Error())
	}

	return validation.NewError("validation_exists", fmt.Sprintf("指定的 %s 不存在", e.field))
}
