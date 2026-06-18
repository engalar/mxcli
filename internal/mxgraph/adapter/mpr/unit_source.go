package mpr

import (
	"github.com/mendixlabs/mxcli/modelsdk"
	"github.com/mendixlabs/mxcli/modelsdk/element"
)

// modelsdkUnit 包装 LoadUnit + Raw() 获取 raw BSON。
// 注意：LoadUnit() 触发 typed codec 解码，但 Raw() 返回的是 codec 解码前的
// 原始 BSON bytes——codec 解码只发生在 Properties()/ChildElement() 等访问上，
// Raw() 不受影响。
type modelsdkUnit struct {
	id       string
	typeName string
	raw      []byte
}

func (u *modelsdkUnit) ID() string          { return u.id }
func (u *modelsdkUnit) TypeName() string    { return u.typeName }
func (u *modelsdkUnit) Raw() []byte         { return u.raw }

// ModelsdkUnitSource 将 *modelsdk.Model 适配为 RawUnitSource。
//
// 为何不直接用 typed 路径？
//   Forms$FormCallArgument 的 typed Properties 未声明 Widget/Widgets 字段，
//   因此 typed 路径无法到达 widget 树。本章节开头有详述。
//
// 性能说明：LoadUnit() 确实触发 typed codec 解码，但 Raw() 返回的是解码前
// 的原始 BSON bytes，codec 的递归展开只发生在调用 Properties()/ChildElement()
// 时（而本 adapter 不用 typed 路径），所以 LoadUnit 的开销仅限于顶层单元解码
// （约 1-2ms/unit），不会递归解码整个 widget 树。
type ModelsdkUnitSource struct {
	Model *modelsdk.Model
}

func (s *ModelsdkUnitSource) Units() []RawUnit {
	infos := s.Model.Units()
	result := make([]RawUnit, 0, len(infos))
	for _, info := range infos {
		elem, err := s.Model.LoadUnit(info.ID)
		if err != nil {
			continue
		}
		result = append(result, &modelsdkUnit{
			id:       string(elem.ID()),
			typeName: elem.TypeName(),
			raw:      []byte(elem.Raw()),
		})
	}
	return result
}

func (s *ModelsdkUnitSource) ResolveModuleName(unitID string) string {
	return s.Model.ResolveModuleName(element.ID(unitID))
}
