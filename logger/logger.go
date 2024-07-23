package logger

import (
	"github.com/anoideaopen/common-component/loggerhlp"
	"github.com/anoideaopen/glog"
	"github.com/anoideaopen/robot/config"
	"github.com/anoideaopen/robot/hlf/hlfprofile"
)

// New - creates and returns logger
func New(cfg *config.Config, hlfProfile *hlfprofile.HlfProfile, appInfoVer string) (glog.Logger, error) {
	l, err := loggerhlp.CreateLogger(cfg.LogType, cfg.LogLevel)
	if err != nil {
		return nil, err
	}

	l = l.With(Labels{
		Version:   appInfoVer,
		UserName:  cfg.UserName,
		OrgName:   hlfProfile.OrgName,
		Component: ComponentMain,
	}.Fields()...)

	return l, nil
}
