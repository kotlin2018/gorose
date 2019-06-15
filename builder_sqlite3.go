package gorose

import (
	"errors"
	"fmt"
	"github.com/gohouse/gorose/across"
	"github.com/gohouse/gorose/utils"
	"github.com/gohouse/t"
	"strconv"
	"strings"
)

type BuilderSqlite3 struct {
	IOrm
	bindParams []interface{}
}

// sqlstr := fmt.Sprintf("SELECT %s%s FROM %s%s%s%s%s%s%s%s",
//		distinct, fields, table, join, where, group, having, order, limit, offset)
// select {distinct} {fields} from {table} {join} {where} {group} {having} {order} {limit} {offset}
func init() {
	NewDriver().Register("sqlite3", &BuilderSqlite3{})
}

func (b *BuilderSqlite3) Option() {

}

func (b *BuilderSqlite3) BuildQuery(o IOrm) (sqlStr string, args []interface{}, err error) {
	//fmt.Println(b.bindParams)
	b.IOrm = o
	join, err := b.BuildJoin()
	if err != nil {
		return
	}
	where, err := b.BuildWhere()
	if err != nil {
		return
	}
	sqlStr = fmt.Sprintf("SELECT %s%s FROM %s%s%s%s%s%s%s%s",
		b.BuildDistinct(), b.BuildFields(), b.BuildTable(), join, where,
		b.BuildGroup(), b.BuildHaving(), b.BuildOrder(), b.BuildLimit(), b.BuildOffset())

	args = b.bindParams
	return
}



// BuildExecut : build execute query string
func (b *BuilderSqlite3) BuildExecute(o IOrm, operType string) (sqlStr string, args []interface{}, err error) {
	// insert : {"name":"fizz, "website":"fizzday.net"} or {{"name":"fizz2", "website":"www.fizzday.net"}, {"name":"fizz", "website":"fizzday.net"}}}
	// update : {"name":"fizz", "website":"fizzday.net"}
	// delete : ...
	b.IOrm = o
	var update, insertkey, insertval, sqlstr string
	if operType != "delete" {
		update, insertkey, insertval = b.buildData()
	}

	where, err := b.BuildWhere()
	if err != nil {
		return
	}
	switch operType {
	case "insert":
		sqlstr = fmt.Sprintf("INSERT INTO %s (%s) VALUES %s", b.BuildTable(), insertkey, insertval)
	case "update":
		if res=="" && ormApi.Sforce==false{
			return sqlstr, errors.New("出于安全考虑, update时where条件不能为空, 如果真的不需要where条件, 请使用force(如: db.xxx.Force().Update())")
		}
		sqlstr = fmt.Sprintf("UPDATE %s SET %s%s", tableName, update, where)
	case "delete":
		if res=="" && ormApi.Sforce==false{
			return sqlstr, errors.New("出于安全考虑, delete时where条件不能为空, 如果真的不需要where条件, 请使用force(如: db.xxx.Force().Delete())")
		}
		sqlstr = fmt.Sprintf("DELETE FROM %s%s", tableName, where)
	}
	//fmt.Println(sqlstr)
	//dba.Reset()

	return sqlstr, nil
}



// buildData : build inert or update data
func (b *BuilderSqlite3) buildData() (string, string, string) {
	// insert
	var dataFields []string
	var dataValues []string
	// update or delete
	var dataObj []string

	data := b.IOrm.GetData()

	switch data.(type) {
	case string:
		dataObj = append(dataObj, data.(string))
	case []map[string]interface{}: // insert multi datas ([]map[string]interface{})
		datas := data.([]map[string]interface{})
		for key, _ := range datas[0] {
			if inArray(key, dataFields) == false {
				dataFields = append(dataFields, key)
			}
		}
		for _, item := range datas {
			var dataValuesSub []string
			for _, key := range dataFields {
				if item[key] == nil {
					dataValuesSub = append(dataValuesSub, "null")
				} else {
					dataValuesSub = append(dataValuesSub, utils.AddSingleQuotes(item[key]))
				}
			}
			dataValues = append(dataValues, "("+strings.Join(dataValuesSub, ",")+")")
		}
	default: // update or insert
		var dataValuesSub []string
		for key, val := range data.(map[string]interface{}) {
			// insert
			dataFields = append(dataFields, key)
			//dataValuesSub = append(dataValuesSub, utils.AddSingleQuotes(val))
			if val == nil {
				dataValuesSub = append(dataValuesSub, "null")
			} else {
				dataValuesSub = append(dataValuesSub, utils.AddSingleQuotes(val))
			}
			// update
			//dataObj = append(dataObj, key+"="+utils.AddSingleQuotes(val))
			if val == nil {
				dataObj = append(dataObj, key+"=null")
			} else {
				dataObj = append(dataObj, key+"="+utils.AddSingleQuotes(val))
			}
		}
		// insert
		dataValues = append(dataValues, "("+strings.Join(dataValuesSub, ",")+")")
	}

	return strings.Join(dataObj, ","), strings.Join(dataFields, ","), strings.Join(dataValues, ",")
}

func (b *BuilderSqlite3) BuildJoin() (string, error) {
	// 用户传入的join参数+join类型
	var join []interface{}
	var returnJoinArr []string
	joinArr := b.GetJoin()

	for _, join = range joinArr {
		var w string
		var ok bool
		// 用户传入 join 的where值, 即第二个参数
		var args []interface{}

		if len(join) != 2 {
			return "", errors.New("join conditions are wrong")
		}

		// 获取真正的用户传入的join参数
		if args, ok = join[1].([]interface{}); !ok {
			return "", errors.New("join conditions are wrong")
		}

		argsLength := len(args)
		switch argsLength {
		case 1: // join字符串 raw
			w = args[0].(string)
		case 2: // join表 + 字符串
			w = args[0].(string) + " ON " + args[1].(string)
		case 4: // join表 + (a字段+关系+a字段)
			w = args[0].(string) + " ON " + args[1].(string) + " " + args[2].(string) + " " + args[3].(string)
		default:
			return "", errors.New("join format error")
		}

		returnJoinArr = append(returnJoinArr, " "+join[0].(string)+" JOIN "+w)
	}

	return strings.Join(returnJoinArr, " "), nil
}

func (b *BuilderSqlite3) BuildWhere() (where string, err error) {
	var beforeParseWhere = b.IOrm.GetWhere()
	where, err = b.parseWhere(b.IOrm)
	b.IOrm.SetWhere(beforeParseWhere)
	return If(where == "", "", " WHERE "+where).(string), err
}

func (b *BuilderSqlite3) BuildDistinct() (dis string) {
	return If(b.IOrm.GetDistinct(), "DISTINCT ", "").(string)
}

func (b *BuilderSqlite3) BuildFields() string {
	return strings.Join(b.IOrm.GetFields(), ",")
}

func (b *BuilderSqlite3) BuildTable() string {
	return b.IOrm.GetTable()
}

func (b *BuilderSqlite3) BuildGroup() string {
	return If(b.IOrm.GetGroup() == "", "", " GROUP BY "+b.IOrm.GetGroup()).(string)
}

func (b *BuilderSqlite3) BuildHaving() string {
	return If(b.IOrm.GetHaving() == "", "", " HAVING "+b.IOrm.GetHaving()).(string)
}

func (b *BuilderSqlite3) BuildOrder() string {
	return If(b.IOrm.GetOrder() == "", "", " ORDER BY "+b.IOrm.GetOrder()).(string)
}

func (b *BuilderSqlite3) BuildLimit() string {
	return If(b.IOrm.GetLimit() == 0, "", " LIMIT "+strconv.Itoa(b.IOrm.GetLimit())).(string)
}

func (b *BuilderSqlite3) BuildOffset() string {
	return If(b.IOrm.GetOffset() == 0, "", " OFFSET "+strconv.Itoa(b.IOrm.GetOffset())).(string)
}

// parseWhere : parse where condition
func  (b *BuilderSqlite3) parseWhere(ormApi IOrm) (string, error) {
	// 取出所有where
	wheres := ormApi.GetWhere()
	// where解析后存放每一项的容器
	var where []string

	for _, args := range wheres {
		// and或者or条件
		var condition string = args[0].(string)
		// 统计当前数组中有多少个参数
		params := args[1].([]interface{})
		paramsLength := len(params)

		switch paramsLength {
		case 3: // 常规3个参数:  {"id",">",1}
			res, err := b.parseParams(params, ormApi)
			if err != nil {
				return res, err
			}
			where = append(where, condition+" "+res)

		case 2: // 常规2个参数:  {"id",1}
			res, err := b.parseParams(params, ormApi)
			if err != nil {
				return res, err
			}
			where = append(where, condition+" "+res)
		case 1: // 二维数组或字符串
			switch paramReal := params[0].(type) {
			case string:
				where = append(where, condition+" ("+paramReal+")")
			case map[string]interface{}: // 一维数组
				var whereArr []string
				for key, val := range paramReal {
					//whereArr = append(whereArr, key+"="+addSingleQuotes(val))
					whereArr = append(whereArr, key+"=?")
					b.bindParams = append(b.bindParams, val)
				}
				where = append(where, condition+" ("+strings.Join(whereArr, " and ")+")")
			case [][]interface{}: // 二维数组
				var whereMore []string
				for _, arr := range paramReal { // {{"a", 1}, {"id", ">", 1}}
					whereMoreLength := len(arr)
					switch whereMoreLength {
					case 3:
						res, err := b.parseParams(arr, ormApi)
						if err != nil {
							return res, err
						}
						whereMore = append(whereMore, res)
					case 2:
						res, err := b.parseParams(arr, ormApi)
						if err != nil {
							return res, err
						}
						whereMore = append(whereMore, res)
					default:
						return "", errors.New("where data format is wrong")
					}
				}
				where = append(where, condition+" ("+strings.Join(whereMore, " and ")+")")
			case func():
				//fmt.Println(b.bindParams)
				// 清空where,给嵌套的where让路,复用这个节点
				ormApi.SetWhere([][]interface{}{})

				// 执行嵌套where放入Database struct
				paramReal()
				// 再解析一遍后来嵌套进去的where
				wherenested, err := b.parseWhere(ormApi)
				if err != nil {
					return "", err
				}
				// 嵌套的where放入一个括号内
				where = append(where, condition+" ("+wherenested+")")
			default:
				return "", errors.New("where data format is wrong")
			}
		}
	}

	return strings.TrimLeft(
		strings.TrimLeft(strings.TrimLeft(
			strings.Trim(strings.Join(where, " "), " "),
			"and"), "or"),
		" "), nil
}

/**
 * 将where条件中的参数转换为where条件字符串
 * example: {"id",">",1}, {"age", 18}
 */
// parseParams : 将where条件中的参数转换为where条件字符串
func (b *BuilderSqlite3) parseParams(args []interface{}, ormApi IOrm) (string, error) {
	paramsLength := len(args)
	argsReal := args

	// 存储当前所有数据的数组
	var paramsToArr []string

	switch paramsLength {
	case 3: // 常规3个参数:  {"id",">",1}
		if !inArray(argsReal[1], ormApi.GetRegex()) {
			return "", errors.New("where parameter is wrong")
		}

		paramsToArr = append(paramsToArr, argsReal[0].(string))
		paramsToArr = append(paramsToArr, argsReal[1].(string))

		switch argsReal[1] {
		case "like", "not like":
			//paramsToArr = append(paramsToArr, addSingleQuotes(argsReal[2]))
			paramsToArr = append(paramsToArr, "?")
			b.bindParams = append(b.bindParams, argsReal[2])
		case "in", "not in":
			var tmp []string
			var ar2 = t.New(argsReal[2]).MapStr()
			//switch argsReal[2].(type) {
			//case []string:
			//	for _, item := range argsReal[2].([]string) {
			//		//tmp = append(tmp, addSingleQuotes(item))
			//		tmp = append(tmp, "?")
			//		b.bindParams = append(b.bindParams, item)
			//	}
			//case []int:
			//	for _, item := range argsReal[2].([]int) {
			//		//tmp = append(tmp, addSingleQuotes(item))
			//		tmp = append(tmp, "?")
			//		b.bindParams = append(b.bindParams, item)
			//	}
			//case []interface{}:
			//	for _, item := range argsReal[2].([]interface{}) {
			//		//tmp = append(tmp, addSingleQuotes(item))
			//		tmp = append(tmp, "?")
			//		b.bindParams = append(b.bindParams, item)
			//	}
			//}
			for _, item := range ar2 {
				tmp = append(tmp, "?")
				b.bindParams = append(b.bindParams, t.New(item).Interface())
			}
			paramsToArr = append(paramsToArr, "("+strings.Join(tmp, ",")+")")
		case "between", "not between":
			//var tmpB []interface{}
			var ar2 = t.New(argsReal[2]).Slice()
			//switch argsReal[2].(type) {
			//case []string:
			//	tmp := argsReal[2].([]string)
			//	tmpB = append(tmpB, tmp[0])
			//	tmpB = append(tmpB, tmp[1])
			//case []int:
			//	tmp := argsReal[2].([]int)
			//	tmpB = append(tmpB, tmp[0])
			//	tmpB = append(tmpB, tmp[1])
			//case []interface{}:
			//	tmp := argsReal[2].([]interface{})
			//	tmpB = append(tmpB, tmp[0])
			//	tmpB = append(tmpB, tmp[1])
			//}
			//paramsToArr = append(paramsToArr, addSingleQuotes(tmpB[0])+" and "+addSingleQuotes(tmpB[1]))
			paramsToArr = append(paramsToArr, "? and ?")
			b.bindParams = append(b.bindParams, ar2[0].Interface())
			b.bindParams = append(b.bindParams, ar2[1].Interface())
		default:
			//paramsToArr = append(paramsToArr, addSingleQuotes(argsReal[2]))
			paramsToArr = append(paramsToArr, "?")
			b.bindParams = append(b.bindParams, argsReal[2])
		}
	case 2:
		paramsToArr = append(paramsToArr, argsReal[0].(string))
		paramsToArr = append(paramsToArr, "=")
		//paramsToArr = append(paramsToArr, addSingleQuotes(argsReal[1]))
		paramsToArr = append(paramsToArr, "?")
		b.bindParams = append(b.bindParams, argsReal[1])
	}

	return strings.Join(paramsToArr, " "), nil
}
