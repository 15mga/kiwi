package dynamo

import (
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type ITable interface {
	Name() string
	AttributeDefinitions() []types.AttributeDefinition
	KeySchema() []types.KeySchemaElement
}

func NewTable(name, hashKey, rangeKey string, attrs map[string]types.ScalarAttributeType) ITable {
	ads := make([]types.AttributeDefinition, 0, len(attrs))
	for name, attr := range attrs {
		ads = append(ads, types.AttributeDefinition{
			AttributeName: &name,
			AttributeType: attr,
		})
	}
	return &table{
		name:         name,
		attributeMap: ads,
		hashAttr:     hashKey,
		rangeAttr:    rangeKey,
	}
}

type table struct {
	name         string
	attributeMap []types.AttributeDefinition
	hashAttr     string
	rangeAttr    string
}

func (t table) Name() string {
	return t.name
}

func (t table) AttributeDefinitions() []types.AttributeDefinition {
	slc := make([]types.AttributeDefinition, 0, len(t.attributeMap))
	for _, attr := range t.attributeMap {
		slc = append(slc, attr)
	}
	return slc
}

func (t table) KeySchema() []types.KeySchemaElement {
	if t.rangeAttr == "" {
		return []types.KeySchemaElement{
			{
				AttributeName: &t.hashAttr,
				KeyType:       types.KeyTypeHash,
			},
		}
	}
	return []types.KeySchemaElement{
		{
			AttributeName: &t.hashAttr,
			KeyType:       types.KeyTypeHash,
		},
		{
			AttributeName: &t.rangeAttr,
			KeyType:       types.KeyTypeRange,
		},
	}
}
