package dynamo

import (
	"context"
	"errors"
	"fmt"
	"github.com/15mga/kiwi"
	"github.com/15mga/kiwi/util"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/smithy-go/logging"
)

var (
	_Client      *dynamodb.Client
	_NameToTable = make(map[string]ITable)
)

func Client() *dynamodb.Client {
	return _Client
}

func ConnLocal(url string) *util.Err {
	cfg, err := config.LoadDefaultConfig(util.Ctx(),
		config.WithRegion("ap-east-1"),
		config.WithEndpointResolverWithOptions(localEndpointResolver{
			url: url,
		}),
		//config.WithCredentialsProvider(localCredentialsProvider{}),
		config.WithLogger(logger{}),
	)
	if err != nil {
		return util.NewErr(util.EcParamsErr, util.M{
			"err": err.Error(),
		})
	}
	_Client = dynamodb.NewFromConfig(cfg)
	return nil
}

func ConnAWS(region string) *util.Err {
	cfg, err := config.LoadDefaultConfig(util.Ctx(),
		config.WithRegion(region),
		config.WithLogger(logger{}),
	)
	if err != nil {
		return util.NewErr(util.EcParamsErr, util.M{
			"err": err.Error(),
		})
	}
	_Client = dynamodb.NewFromConfig(cfg)
	return nil
}

type localEndpointResolver struct {
	url string
}

func (l localEndpointResolver) ResolveEndpoint(service, region string, options ...interface{}) (aws.Endpoint, error) {
	return aws.Endpoint{
		URL: l.url,
	}, nil
}

type localCredentialsProvider struct {
}

func (l localCredentialsProvider) Retrieve(ctx context.Context) (aws.Credentials, error) {
	return aws.Credentials{
		AccessKeyID: "dummy", SecretAccessKey: "dummy", SessionToken: "dummy",
		Source: "Hard-coded credentials; values are irrelevant for local DynamoDB",
	}, nil
}

type logger struct {
}

func (l logger) Logf(classification logging.Classification, format string, v ...interface{}) {
	switch classification {
	case logging.Debug:
		kiwi.Debug(fmt.Sprintf(format, v...), nil)
	case logging.Warn:
		kiwi.Warn(util.NewErr(util.EcDbErr, util.M{
			"error": fmt.Sprintf(format, v...),
		}))
	}
}

func BindTable(table ITable) {
	_NameToTable[table.Name()] = table
}

func CreateTable(tableName string) *util.Err {
	table, ok := _NameToTable[tableName]
	if !ok {
		return util.NewErr(util.EcNotExist, util.M{
			"table": tableName,
		})
	}
	return creatTable(table)
}

func creatTable(table ITable) *util.Err {
	params := &dynamodb.CreateTableInput{
		TableName:            aws.String(table.Name()),
		AttributeDefinitions: table.AttributeDefinitions(),
		KeySchema:            table.KeySchema(),
	}
	_, e := _Client.CreateTable(util.Ctx(), params)
	if e != nil {
		return util.WrapErr(util.EcDbErr, e)
	}
	return nil
}

func MigrateTables() {
	for name, table := range _NameToTable {
		ad := table.AttributeDefinitions()
		if ad == nil || len(ad) == 0 {
			continue
		}
		exist, err := IsTableExist(name)
		if err != nil {
			kiwi.Error(err)
			continue
		}
		if exist {
			continue
		}
		err = creatTable(table)
		if err == nil {
			kiwi.Error(err)
		}
	}
}

func IsTableExist(table string) (bool, *util.Err) {
	_, e := _Client.DescribeTable(util.Ctx(), &dynamodb.DescribeTableInput{TableName: aws.String(table)})
	if e != nil {
		var notFoundEx *types.ResourceNotFoundException
		if errors.As(e, &notFoundEx) {
			return false, nil
		}
		return false, util.WrapErr(util.EcDbErr, e)
	}
	return true, nil
}

type MAV map[string]types.AttributeValue

func mToAv(data util.M, avs MAV) {
	for attr, val := range data {
		av, e := attributevalue.Marshal(val)
		if e != nil {
			kiwi.Warn(util.WrapErr(util.EcMarshallErr, e))
			continue
		}
		avs[attr] = av
	}
}

func GetEntity[T any](table string, filter util.M, consistent bool, entity T, projectAttrs ...string) *util.Err {
	mav := make(MAV)
	mToAv(filter, mav)
	params := &dynamodb.GetItemInput{
		Key:            mav,
		TableName:      aws.String(table),
		ConsistentRead: &consistent,
	}
	if len(projectAttrs) > 0 {
		pb := expression.ProjectionBuilder{}
		for _, attr := range projectAttrs {
			pb.AddNames(expression.Name(attr))
		}
		expr, e := expression.NewBuilder().WithProjection(pb).Build()
		if e != nil {
			return util.WrapErr(util.EcParamsErr, e)
		}
		params.ExpressionAttributeNames = expr.Names()
		params.ProjectionExpression = expr.Projection()
	}
	res, e := _Client.GetItem(util.Ctx(), params)
	if e != nil {
		return util.WrapErr(util.EcDbErr, e)
	}
	e = attributevalue.UnmarshalMap(res.Item, entity)
	return util.WrapErr(util.EcDbErr, e)
}

func PutNewEntity(table string, entity util.M, unique ...string) *util.Err {
	item, e := attributevalue.MarshalMap(entity)
	if e != nil {
		return util.WrapErr(util.EcMarshallErr, e)
	}
	ce := ""
	i := 0
	c := len(unique) - 1
	for _, key := range unique {
		ce += "attribute_not_exists(" + key + ")"
		if i < c {
			ce += " AND "
		}
	}
	params := &dynamodb.PutItemInput{
		TableName:           aws.String(table),
		Item:                item,
		ConditionExpression: &ce,
	}
	_, e = _Client.PutItem(util.Ctx(), params)
	return util.WrapErr(util.EcDbErr, e)
}

func PutOrReplaceEntity(table string, entity any) *util.Err {
	item, e := attributevalue.MarshalMap(entity)
	if e != nil {
		return util.WrapErr(util.EcMarshallErr, e)
	}
	params := &dynamodb.PutItemInput{
		TableName: aws.String(table),
		Item:      item,
	}
	_, e = _Client.PutItem(util.Ctx(), params)
	return util.WrapErr(util.EcDbErr, e)
}

type UpdateResult map[string]map[string]any

func buildUpdateParams(table string, params *dynamodb.UpdateItemInput, filter, data util.M) *util.Err {
	l := len(data)
	if l == 0 {
		return util.NewErr(util.EcParamsErr, util.M{
			"error": "attrs length is zero",
		})
	}
	mav := make(MAV)
	mToAv(filter, mav)
	ub := expression.UpdateBuilder{}
	for attr, val := range data {
		ub.Set(expression.Name(attr), expression.Value(val))
	}
	expr, e := expression.NewBuilder().WithUpdate(ub).Build()
	if e != nil {
		return util.WrapErr(util.EcParamsErr, e)
	}
	params.Key = mav
	params.TableName = aws.String(table)
	params.ExpressionAttributeNames = expr.Names()
	params.ExpressionAttributeValues = expr.Values()
	params.UpdateExpression = expr.Update()
	return nil
}

func UpdateEntity(table string, filter, data util.M) *util.Err {
	params := &dynamodb.UpdateItemInput{}
	err := buildUpdateParams(table, params, filter, data)
	if err != nil {
		return err
	}
	_, e := _Client.UpdateItem(util.Ctx(), params)
	return util.WrapErr(util.EcDbErr, e)
}

func UpdateEntityWithResult(table string, filter, data util.M, resultValue types.ReturnValue, result UpdateResult) *util.Err {
	params := &dynamodb.UpdateItemInput{}
	err := buildUpdateParams(table, params, filter, data)
	if err != nil {
		return err
	}
	params.ReturnValues = resultValue
	res, e := _Client.UpdateItem(util.Ctx(), params)
	if e != nil {
		return util.WrapErr(util.EcDbErr, e)
	}
	e = attributevalue.UnmarshalMap(res.Attributes, &result)
	return util.WrapErr(util.EcUnmarshallErr, e)
}

func getQueryEqualExpr(table string, filter util.M, params *dynamodb.QueryInput, projectAttrs []string) *util.Err {
	b := expression.NewBuilder()
	for attr, val := range filter {
		kcb := expression.Key(attr).Equal(expression.Value(val))
		b.WithKeyCondition(kcb)
	}

	hasProject := buildProjection(&b, projectAttrs)
	expr, e := b.Build()
	if e != nil {
		return util.WrapErr(util.EcParamsErr, e)
	}

	params.ExpressionAttributeNames = expr.Names()
	if hasProject {
		params.ProjectionExpression = expr.Projection()
	}

	params.TableName = aws.String(table)
	params.ExpressionAttributeNames = expr.Names()
	params.ExpressionAttributeValues = expr.Values()
	params.KeyConditionExpression = expr.KeyCondition()
	return nil
}

func buildProjection(builder *expression.Builder, projectAttrs []string) bool {
	if projectAttrs == nil || len(projectAttrs) == 0 {
		return false
	}
	pb := expression.ProjectionBuilder{}
	for _, attr := range projectAttrs {
		pb.AddNames(expression.Name(attr))
	}
	builder.WithProjection(pb)
	return true
}

func getQueryBetweenExpr(table string, params *dynamodb.QueryInput, key string, start, end any, limit int32,
	forward bool, projectAttrs []string) *util.Err {
	b := expression.NewBuilder()
	kcb := expression.Key(key).Between(expression.Value(start), expression.Value(end))
	b.WithKeyCondition(kcb)

	hasProject := buildProjection(&b, projectAttrs)
	expr, e := b.Build()
	if e != nil {
		return util.WrapErr(util.EcParamsErr, e)
	}

	params.ExpressionAttributeNames = expr.Names()
	if hasProject {
		params.ProjectionExpression = expr.Projection()
	}

	params.TableName = aws.String(table)
	params.ExpressionAttributeNames = expr.Names()
	params.ExpressionAttributeValues = expr.Values()
	params.KeyConditionExpression = expr.KeyCondition()
	params.ScanIndexForward = &forward
	if limit > 0 {
		params.Limit = &limit
	}
	return nil
}

func QueryEntities[T any](table string, filter util.M, items *[]T, projectAttrs ...string) *util.Err {
	params := &dynamodb.QueryInput{}
	err := getQueryEqualExpr(table, filter, params, projectAttrs)
	if err != nil {
		return err
	}

	res, e := _Client.Query(util.Ctx(), params)
	if e != nil {
		return util.WrapErr(util.EcDbErr, e)
	}
	e = attributevalue.UnmarshalListOfMaps(res.Items, items)
	return util.WrapErr(util.EcUnmarshallErr, e)
}

func QueryEntitiesBetween[T any](table, key string, start, end any, limit int32,
	forward bool, items *[]T, projectAttrs ...string) *util.Err {
	params := &dynamodb.QueryInput{}
	err := getQueryBetweenExpr(table, params, key, start, end, limit, forward, projectAttrs)
	if err != nil {
		return err
	}

	res, e := _Client.Query(util.Ctx(), params)
	if e != nil {
		return util.WrapErr(util.EcDbErr, e)
	}
	e = attributevalue.UnmarshalListOfMaps(res.Items, items)
	return util.WrapErr(util.EcUnmarshallErr, e)
}
