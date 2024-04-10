package nerrors

import "github.com/anoideaopen/common-component/errorshlp"

const (
	ErrTypeHlf      errorshlp.ErrType = "hlf"
	ErrTypeRedis    errorshlp.ErrType = "redis"
	ErrTypeParsing  errorshlp.ErrType = "parsing"
	ErrTypeInternal errorshlp.ErrType = "internal"

	ComponentStorage   errorshlp.ComponentName = "storage"
	ComponentCollector errorshlp.ComponentName = "collector"
	ComponentExecutor  errorshlp.ComponentName = "executor"
	ComponentRobot     errorshlp.ComponentName = "robot"
	ComponentParser    errorshlp.ComponentName = "parser"
	ComponentBatch     errorshlp.ComponentName = "batch"
)
