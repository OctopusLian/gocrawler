package sqlstorage

import (
	"encoding/json"
	"errors"
	"go.uber.org/zap"
	"gocrawler/engine"
	"gocrawler/spider"
	"gocrawler/sqldb"
)

type SQLStore struct {
	dataDocker []*spider.DataCell //分批输出结果缓存
	db         sqldb.DBer
	Table      map[string]struct{}
	options
}

func New(opts ...Option) (*SQLStore, error) {
	options := defaultOptions
	for _, opt := range opts {
		opt(&options)
	}
	s := &SQLStore{}
	s.options = options
	s.Table = make(map[string]struct{})
	var err error
	s.db, err = sqldb.New(
		sqldb.WithConnURL(s.sqlURL),
		sqldb.WithLogger(s.logger),
	)
	if err != nil {
		return nil, err
	}

	return s, nil
}

func (s *SQLStore) Save(dataCells ...*spider.DataCell) error {
	for _, cell := range dataCells {
		name := cell.GetTableName()
		if _, ok := s.Table[name]; !ok {
			// 获取当前数据的表字段与字段类型
			columnNames := getFields(cell)

			err := s.db.CreateTable(sqldb.TableData{
				TableName:   name,
				ColumnNames: columnNames,
				AutoKey:     true,
			})
			if err != nil {
				s.logger.Error("create table falied", zap.Error(err))
				continue
			}
			s.Table[name] = struct{}{}
		}
		if len(s.dataDocker) >= s.BatchCount {
			// 如果缓冲区已经满了，则调用 SqlStore.Flush() 方法批量插入数据
			s.Flush()
		}
		// 如果当前的数据小于 s.BatchCount，则将数据放入到缓存中直接返回 (用缓冲区批量插入数据库可以提高程序的性能)
		s.dataDocker = append(s.dataDocker, cell)
	}
	return nil
}

func getFields(cell *spider.DataCell) []sqldb.Field {
	taskName := cell.Data["Task"].(string)
	ruleName := cell.Data["Rule"].(string)
	fields := engine.GetFields(taskName, ruleName)

	var columnNames []sqldb.Field
	for _, field := range fields {
		columnNames = append(columnNames, sqldb.Field{
			Title: field,
			Type:  "MEDIUMTEXT",
		})
	}
	columnNames = append(columnNames,
		sqldb.Field{Title: "URL", Type: "VARCHAR(255)"},
		sqldb.Field{Title: "Time", Type: "VARCHAR(255)"},
	)
	return columnNames
}

func (s *SQLStore) Flush() error {
	if len(s.dataDocker) == 0 {
		return nil
	}
	args := make([]interface{}, 0)

	var ruleName string
	var taskName string
	var ok bool
	for _, datacell := range s.dataDocker {
		if ruleName, ok = datacell.Data["Rule"].(string); !ok {
			return errors.New("no rule field")
		}

		if taskName, ok = datacell.Data["Task"].(string); !ok {
			return errors.New("no task field")
		}
		fields := engine.GetFields(taskName, ruleName)
		data := datacell.Data["Data"].(map[string]interface{})
		value := []string{}
		for _, field := range fields {
			v := data[field]
			switch v.(type) {
			case nil:
				value = append(value, "")
			case string:
				value = append(value, v.(string))
			default:
				j, err := json.Marshal(v)
				if err != nil {
					value = append(value, "")
				} else {
					value = append(value, string(j))
				}
			}
		}

		if v, ok := datacell.Data["URL"].(string); ok {
			value = append(value, v)
		}
		if v, ok := datacell.Data["Time"].(string); ok {
			value = append(value, v)
		}
		for _, v := range value {
			args = append(args, v)
		}
	}

	return s.db.Insert(sqldb.TableData{
		TableName:   s.dataDocker[0].GetTableName(),
		ColumnNames: getFields(s.dataDocker[0]),
		Args:        args,
		DataCount:   len(s.dataDocker),
	})
}
