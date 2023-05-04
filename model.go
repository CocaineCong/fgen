package main

import (
	"bytes"
	"context"
	"fmt"
	"go/format"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/go-sql-driver/mysql"
	"github.com/gogf/gf/database/gdb"
	"github.com/gogf/gf/frame/g"
	"github.com/gogf/gf/os/gcmd"
	"github.com/gogf/gf/os/gfile"
	"github.com/gogf/gf/os/glog"
	"github.com/gogf/gf/text/gregex"
	"github.com/gogf/gf/text/gstr"
	"github.com/olekukonko/tablewriter"
	"gopkg.in/yaml.v2"
)

type Config struct {
	Mysql Mysql `yaml:"mysql"`
}

type ConfigMap struct {
	Mysql map[string]Mysql `yaml:"mysql"`
}

type Mysql struct {
	Dialect  string `yaml:"dialect"`
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	DBName   string `yaml:"dbName"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Charset  string `yaml:"charset"`
}

func GenModelV2(ctx context.Context, dsn, genPath, genPkg, configPath, key string, tables ...string) error {
	if genPath == "" {
		// 外层能保证不为空
		genPath = "_output/model"
	}

	if genPkg == "" {
		genPkg = filepath.Base(genPath) // default:db
	}

	var (
		mysqlInfo *Mysql
		err       error
		dbNode    gdb.ConfigNode
	)

	if dsn != "" {
		// 解析mysql dsn
		cfg, err := mysql.ParseDSN(dsn)
		if err != nil {
			return err
		}
		dbNode = gdb.ConfigNode{
			Host: strings.Split(cfg.Addr, ":")[0],
			Port: strings.Split(cfg.Addr, ":")[1],
			User: strings.Split(cfg.Passwd, ":")[0],
			Pass: strings.Split(cfg.Passwd, ":")[1],
			Name: cfg.DBName,
			Type: "mysql",
		}
	} else {
		// 从配置文件中读取
		mysqlInfo, err = getMysqlConfig(configPath, key)
		if err != nil {
			return err
		}
		dbNode = gdb.ConfigNode{
			Host:    mysqlInfo.Host,
			Port:    mysqlInfo.Port,
			User:    mysqlInfo.User,
			Pass:    mysqlInfo.Password,
			Name:    mysqlInfo.DBName,
			Type:    mysqlInfo.Dialect,
			Charset: mysqlInfo.Charset,
		}
	}

	gdb.SetConfig(gdb.Config{
		"default": gdb.ConfigGroup{
			dbNode,
		},
	})
	db, err := gdb.New("default")
	if err != nil {
		glog.Fatal("database initialization failed")
		return err
	}

	if err := gfile.Mkdir(genPath); err != nil {
		glog.Fatal("mkdir for generating path:%s failed: %v", genPath, err)
	}

	if len(tables) == 0 {
		tables, err = db.Tables(context.Background())
		if err != nil {
			glog.Fatal("get mysql info all tables")
		}
	}

	for _, table := range tables {
		table = strings.TrimSpace(table)
		if table == "" {
			continue
		}
		genModelContentFile(ctx, genPkg, db, table, genPath)
	}
	glog.Print("done!")
	return nil
}

func getMysqlConfig(configPath, key string) (*Mysql, error) {
	file, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, err
	}
	var config Config
	var configMap ConfigMap
	if key != "" {
		err = yaml.Unmarshal(file, &configMap)
		if err != nil {
			return nil, err
		}
		if c, ok := configMap.Mysql[key]; ok {
			return &c, nil
		}
		return nil, fmt.Errorf("key not found")
	}
	err = yaml.Unmarshal(file, &config)
	if err != nil {
		return nil, err
	}
	return &config.Mysql, nil
}

// 生成结构体对象
func genStructDefinition(camelName string, fieldMap map[string]*gdb.TableField) string {
	buffer := bytes.NewBuffer(nil)
	array := make([][]string, len(fieldMap))
	for _, field := range fieldMap {
		array[field.Index] = genStructField(field)
	}
	tw := tablewriter.NewWriter(buffer)
	tw.SetBorder(false)
	tw.SetRowLine(false)
	tw.SetAutoWrapText(false)
	tw.SetCenterSeparator("")
	tw.AppendBulk(array)
	tw.Render()

	stContent := buffer.String()
	stContent = gstr.Replace(stContent, " #", "")
	stContent = gstr.Replace(stContent, " |", "")
	buffer.Reset()
	buffer.WriteString("type ")
	buffer.WriteString(camelName + " struct{\n")
	buffer.WriteString(stContent)
	buffer.WriteString("}")
	return buffer.String()
}

// 生成结构体字段
func genStructField(field *gdb.TableField) []string {
	var typeName, ormTag, comment string
	t, _ := gregex.ReplaceString(`\(.+\)`, "", field.Type)
	t = strings.Split(gstr.Trim(t), " ")[0]
	t = gstr.ToLower(t)
	switch t {
	case "binary", "varbinary", "blob", "tinyblob", "mediumblob", "longblob":
		typeName = "[]byte"
	case "bit", "int", "tinyint", "small_int", "smallint", "medium_int", "mediumint":
		typeName = "int"
		if gstr.Contains(field.Name, "id") {
			typeName = "int64"
		}
	case "big_int", "bigint":
		typeName = "int64"
		// 对Id再单独处理一下
		if gstr.Contains(field.Name, "id") {
			typeName = "int64"
		}
	case "float", "double", "decimal":
		typeName = "time.Time"
	case "datetime", "date", "time":
		typeName = "time.Time"

	default:
		switch {
		case strings.Contains(t, "int"):
			typeName = "int"
		case strings.Contains(t, "text") || strings.Contains(t, "char"):
			typeName = "string"
		case strings.Contains(t, "float") || strings.Contains(t, "double"):
			typeName = "float64"
		case strings.Contains(t, "bool"):
			typeName = "bool"
		case strings.Contains(t, "binary") || strings.Contains(t, "blob"):
			typeName = "[]byte"
		case strings.Contains(t, "date") || strings.Contains(t, "time"):
			typeName = "time.Time"
		default:
			typeName = "string"
		}
	}
	// 对时间再单独处理一下
	if gstr.ContainsI(typeName, "int") {
		if gstr.ContainsI(field.Name, "time") ||
			gstr.ContainsI(field.Name, "create") ||
			gstr.ContainsI(field.Name, "update") {
			typeName = "int64"
		}
	}

	ormTag = field.Name
	if gstr.ContainsI(field.Key, "pri") {
		ormTag = ormTag + ";primary_key;"
	}
	comment = gstr.ReplaceIByArray(field.Comment, g.SliceStr{
		"\n", "",
		"\r", "",
	})
	comment = gstr.Trim(comment)

	gorm := fmt.Sprintf("`"+`gorm:"column:%s"`+"`", ormTag)
	as := []string{
		"   #" + gstr.CaseCamel(field.Name),
		" #" + typeName,
		" #" + gorm,
	}

	if comment != "" {
		as = append(as, " #"+fmt.Sprintf("// %s", comment))
	}

	return as
}

func genModelContentFile(ctx context.Context, genPkg string, db gdb.DB, table, folderPath string) {
	fieldMap, err := db.TableFields(ctx, table)
	if err != nil {
		glog.Fatal("fetching tables fields failed for table: %s :\n %v", table, err)
	}
	variable := gstr.TrimLeftStr(table, ",")
	camelName := gstr.CaseCamel(variable)
	modelName := fmt.Sprintf("%sModel", camelName)
	structDefine := genStructDefinition(modelName, fieldMap)

	fileName := gstr.Trim(gstr.CaseSnake(variable), "-_.")
	path := gfile.Join(folderPath, fileName+".go")

	if !gfile.IsEmpty(path) {
		s := gcmd.Scanf("the '%s' is exist, files might be overwrote, continue?[y/n]:", path)
		if strings.EqualFold(s, "n") {
			return
		}
	}
	timePackage := ""
	if gstr.ContainsI(structDefine, "time.Time") {
		timePackage = `"time"`
	}
	entityContent := gstr.ReplaceByMap(modelTemplate, g.MapStrStr{
		"{package}":         genPkg,
		"{TimePackage}":     timePackage,
		"{TplTableName}":    table,
		"{TplModelName}":    modelName,
		"{TplDaoName}":      gstr.CaseCamelLower(camelName) + "Dao",
		"{TplUpperDaoName}": camelName + "Dao",
		"{TplStructDefine}": structDefine,
		"{TplStructReport}": structDefine,
	})

	bts, err := format.Source([]byte(entityContent))
	if err != nil {
		glog.Fatalf("fmt err:%v", err)
	}

	if err := gfile.PutContents(path, string(bts)); err != nil {
		glog.Fatalf("writing content to %s failed:%v", path, err)
	} else {
		glog.Print("generated:", path)
	}
}

const modelTemplate = `
package {package}

import (
	gormv2 "gorm.io/gorm"
	{TimePackage}
)

{TplStructDefine}

func (*{TplModelName}) TableName() string{
	return "{TplTableName}"
}

type {TplDaoName} struct{
	db *gormv2.DB
}

func New{TplUpperDaoName}(db *gormv2.DB) *{TplDaoName}{
	return &{TplDaoName}{
		db:db,
	}
}

func (s *{TplDaoName}) Get(in *{TplModelName}) (*{TplModelName},error) {
	var r {TplModelName}
	err := s.db.Where(in).Find(&r).Error
	return &r,err
}

func (s *{TplDaoName}) List(in *{TplModelName}) ([]*{TplModelName},error) {
	var r []*{TplModelName}
	err := s.db.Where(in).Find(&r).Error
	return r,err
}

func (s *{TplDaoName}) Create(in *{TplModelName}) error {
	return s.db.Create(in).Error
}

func (s *{TplDaoName}) Update(in *{TplModelName}) error {
	return s.db.Model(&{TplModelName}{}).Updates(in).Error
}
`
