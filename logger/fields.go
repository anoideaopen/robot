package logger

import "github.com/anoideaopen/glog"

type Labels struct {
	Version   string
	UserName  string
	OrgName   string
	Component ComponentName
	ChName    string
	DstChName string
	SrcChName string
}

type ComponentName string

var (
	ComponentMain      ComponentName = "main"
	ComponentStorage   ComponentName = "storage"
	ComponentCollector ComponentName = "collector"
	ComponentExecutor  ComponentName = "executor"
	ComponentRobot     ComponentName = "robot"
	ComponentParser    ComponentName = "parser"
	ComponentBatch     ComponentName = "batch"
)

func (l Labels) Fields() (fields []glog.Field) {
	if l.Version != "" {
		fields = append(fields, glog.Field{K: "version", V: l.Version})
	}
	if l.UserName != "" {
		fields = append(fields, glog.Field{K: "user", V: l.UserName})
	}
	if l.OrgName != "" {
		fields = append(fields, glog.Field{K: "org", V: l.OrgName})
	}
	if l.Component != "" {
		fields = append(fields, glog.Field{K: "component", V: l.Component})
	}
	if l.ChName != "" {
		fields = append(fields, glog.Field{K: "channel", V: l.ChName})
	}
	if l.DstChName != "" {
		fields = append(fields, glog.Field{K: "dstChannel", V: l.DstChName})
	}
	if l.SrcChName != "" {
		fields = append(fields, glog.Field{K: "srcChannel", V: l.SrcChName})
	}
	return
}
