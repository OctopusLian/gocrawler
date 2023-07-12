package collector

type DataCell struct {
	Data map[string]interface{}
}

func (d *DataCell) GetTableName() string {
	return d.Data["Task"].(string)
}

func (d *DataCell) GetTaskName() string {
	return d.Data["Task"].(string)
}

type Storage interface {
	Save(datas ...*DataCell) error // 任何实现了 Save 方法的后端引擎都可以存储数据
}
