package logger

import "github.com/newity/glog"

// Labels is a struct with labels for logger
type Labels struct {
	// Version is a version of the application
	Version string
	// UserName is a name of the user
	UserName string
	// OrgName is a name of the organization
	OrgName string
	// Component is a name of the component
	Component ComponentName
	// ChName is a name of the channel
	ChName string
	// DstChName is a name of the destination channel
	DstChName string
	// SrcChName is a name of the source channel
	SrcChName string
}

// ComponentName is a name of the component
type ComponentName string

var (
	// ComponentMain is a name of the main component
	ComponentMain ComponentName = "main"
	// ComponentStorage is a name of the storage component
	ComponentStorage ComponentName = "storage"
	// ComponentCollector is a name of the collector component
	ComponentCollector ComponentName = "collector"
	// ComponentExecutor is a name of the executor component
	ComponentExecutor ComponentName = "executor"
	// ComponentRobot is a name of the robot component
	ComponentRobot ComponentName = "robot"
	// ComponentParser is a name of the parser component
	ComponentParser ComponentName = "parser"
	// ComponentBatch is a name of the batch component
	ComponentBatch ComponentName = "batch"
)

// Fields returns fields for logger
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
