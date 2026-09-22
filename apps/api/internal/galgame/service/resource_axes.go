package service

import (
	"kun-galgame-api/internal/galgame/resourcevocab"
	"kun-galgame-api/pkg/errors"
)

type resourceAxes struct {
	Type         string
	Title        string
	VersionLabel string
	Language     string
	Platform     string
	Languages    resourcevocab.Keys
	Platforms    resourcevocab.Keys
	Runtimes     resourcevocab.Keys
}

func parseResourceAxes(
	typ, title, version string,
	languages, platforms, runtimes []string,
	legacyLang, legacyPlat string,
) (resourceAxes, *errors.AppError) {
	if !resourcevocab.IsType(typ) {
		return resourceAxes{}, errors.ErrBadRequest("请选择正确的资源类型")
	}
	typ = resourcevocab.CompatType(typ)
	t, ok := resourcevocab.Title(title)
	if !ok {
		return resourceAxes{}, errors.ErrBadRequest("标题不超过 200 字，且必须是单行")
	}
	v, ok := resourcevocab.VersionLabel(version)
	if !ok {
		return resourceAxes{}, errors.ErrBadRequest("适配版本不超过 64 字，且必须是单行")
	}
	langs, ok := resourcevocab.Languages(languages)
	if !ok {
		return resourceAxes{}, errors.ErrBadRequest("请选择正确的资源语言")
	}
	if len(langs) == 0 {
		langs = resourcevocab.LegacyLanguage(legacyLang)
	}
	if len(langs) == 0 {
		return resourceAxes{}, errors.ErrBadRequest("请选择正确的资源语言")
	}
	plats, ok := resourcevocab.Platforms(platforms)
	if !ok {
		return resourceAxes{}, errors.ErrBadRequest("请选择正确的资源平台")
	}
	runs, ok := resourcevocab.Runtimes(runtimes)
	if !ok {
		return resourceAxes{}, errors.ErrBadRequest("请选择正确的运行环境")
	}
	if len(plats) == 0 && len(runs) == 0 {
		plats, runs = resourcevocab.LegacyPlatform(legacyPlat)
	}
	if len(plats) == 0 && legacyPlat != "emulator" {
		return resourceAxes{}, errors.ErrBadRequest("请选择正确的资源平台")
	}
	if resourcevocab.HasRuntimeAxis(typ) && len(runs) == 0 {
		return resourceAxes{}, errors.ErrBadRequest("请选择运行环境")
	}
	if !resourcevocab.HasRuntimeAxis(typ) {
		runs = resourcevocab.Keys{}
	}
	return resourceAxes{
		Type:         typ,
		Title:        t,
		VersionLabel: v,
		Language:     resourcevocab.CompatLanguage(langs),
		Platform:     resourcevocab.CompatPlatform(plats, runs),
		Languages:    langs,
		Platforms:    plats,
		Runtimes:     runs,
	}, nil
}
