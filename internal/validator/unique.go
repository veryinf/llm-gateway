package validator

import (
	"fmt"
	"llm-gateway/internal/core"
	"log/slog"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/samber/lo"
	"gorm.io/gorm"
)

type whereCondition struct {
	query any
	args  []any
}
type UniqueRule struct {
	model   any
	field   string
	pkField string
	pkValue any
	wheres  []whereCondition
	err     error
}

func Unique(model any, field string) UniqueRule {
	return UniqueRule{
		model:   model,
		field:   field,
		pkField: "",
		pkValue: nil,
		wheres:  []whereCondition{},
	}
}

func (e UniqueRule) ExceptPk(pkValue any) UniqueRule {
	if e.pkField == "" {
		migrator := core.DB.Model(e.model).Migrator()
		if !migrator.HasTable(e.model) {
			slog.Error("验证器配置错误: 表不存在", "model", fmt.Sprintf("%T", e.model))
			e.err = fmt.Errorf("表不存在: %T", e.model)
			return e
		}
		columns, _ := migrator.ColumnTypes(e.model)
		primaryKeys := lo.Filter(columns, func(c gorm.ColumnType, i int) bool { p, _ := c.PrimaryKey(); return p })
		if len(primaryKeys) != 1 {
			slog.Error("验证器配置错误: 仅支持单主键", "model", fmt.Sprintf("%T", e.model))
			e.err = fmt.Errorf("仅支持单主键的表: %T", e.model)
			return e
		}
		e.pkField = primaryKeys[0].Name()
	}
	e.pkValue = pkValue
	return e
}

func (e UniqueRule) Where(query any, args ...any) UniqueRule {
	e.wheres = append(e.wheres, whereCondition{query: query, args: args})
	return e
}

func (e UniqueRule) Validate(value any) error {
	if e.err != nil {
		return validation.NewError("validation_config", e.err.Error())
	}
	value, isNil := validation.Indirect(value)
	if isNil || validation.IsEmpty(value) {
		return nil
	}

	var count int64
	query := core.DB.Model(e.model).Where(e.field+"=?", value)
	if e.pkValue != nil {
		query = query.Where(e.pkField+"!=?", e.pkValue)
	}
	for _, where := range e.wheres {
		query = query.Where(where.query, where.args...)
	}
	err := query.Limit(1).Count(&count).Error
	if count > 0 {
		return validation.NewError("validation_unique", fmt.Sprintf("字段{%s}中已存在 %s", e.field, value))
	} else {
		if err != nil {
			return validation.NewError("validation_unique", "判断唯一字段出错, 请检查详情: "+err.Error())
		} else {
			return nil
		}
	}
}
