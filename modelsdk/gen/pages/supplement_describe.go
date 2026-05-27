// SPDX-License-Identifier: Apache-2.0

package pages

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// BSONArrayItems 将 raw BSON 字段统一转换为 []any。
//
// MongoDB Go 驱动在解码 BSON 数组时可能返回 []any 或 primitive.A，
// 两者底层一致，但 Go 类型不同。本函数让 executor 调用方无需直接
// 依赖 primitive 包即可处理两种形式。
//
// 返回的第二个值表示输入是否为可识别的数组类型。
func BSONArrayItems(v any) ([]any, bool) {
	switch arr := v.(type) {
	case []any:
		return arr, true
	case primitive.A:
		return []any(arr), true
	default:
		return nil, false
	}
}

// BSONBinaryData 将 raw BSON 字段中的 primitive.Binary 提取为 []byte。
// 如果输入不是 primitive.Binary，返回 (nil, false)。
//
// 调用方应优先用类型 switch 处理 string / []byte 这些已是 Go 原生
// 类型的情况，再用本函数处理 BSON 专有的 Binary 包装。
func BSONBinaryData(v any) ([]byte, bool) {
	b, ok := v.(primitive.Binary)
	if !ok {
		return nil, false
	}
	return b.Data, true
}
