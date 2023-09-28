package nerrors

import "github.com/atomyze-foundation/common-component/errorshlp"

const (
	// ErrTypeHlf is a type of error when error is from hlf
	ErrTypeHlf errorshlp.ErrType = "hlf"
	// ErrTypeRedis is a type of error when error is from redis
	ErrTypeRedis errorshlp.ErrType = "redis"
	// ErrTypeParsing is a type of error when error is from parsing
	ErrTypeParsing errorshlp.ErrType = "parsing"
	// ErrTypeInternal is a type of error when error is internal
	ErrTypeInternal errorshlp.ErrType = "internal"

	// ComponentStorage is a component name for storage
	ComponentStorage errorshlp.ComponentName = "storage"
	// ComponentCollector is a component name for collector
	ComponentCollector errorshlp.ComponentName = "collector"
	// ComponentExecutor is a component name for executor
	ComponentExecutor errorshlp.ComponentName = "executor"
	// ComponentRobot is a component name for robot
	ComponentRobot errorshlp.ComponentName = "robot"
	// ComponentParser is a component name for parser
	ComponentParser errorshlp.ComponentName = "parser"
	// ComponentBatch is a component name for batch
	ComponentBatch errorshlp.ComponentName = "batch"
)
