package bombsweeper

import (
	"embed"

	"fyne.io/fyne/v2"
)

//go:embed assets/*.svg
var assetFiles embed.FS

func asset(name string) fyne.Resource {
	data, err := assetFiles.ReadFile("assets/" + name)
	if err != nil {
		panic(err)
	}
	return fyne.NewStaticResource(name, data)
}

var (
	resourceClosedPng        = asset("closed.svg")
	resourceFaceLosePng      = asset("face_lose.svg")
	resourceFacePressedPng   = asset("face_pressed.svg")
	resourceFaceUnpressedPng = asset("face_unpressed.svg")
	resourceFaceWinPng       = asset("face_win.svg")
	resourceFlagPng          = asset("flag.svg")
	resourceMinePng          = asset("mine.svg")
	resourceMineRedPng       = asset("mine_red.svg")
	resourceType0Png         = asset("type0.svg")
	resourceType1Png         = asset("type1.svg")
	resourceType2Png         = asset("type2.svg")
	resourceType3Png         = asset("type3.svg")
	resourceType4Png         = asset("type4.svg")
	resourceType5Png         = asset("type5.svg")
	resourceType6Png         = asset("type6.svg")
	resourceType7Png         = asset("type7.svg")
	resourceType8Png         = asset("type8.svg")
)
