package ui

import (
	"example/zerochat/client/types"
	"image/color"

	"gioui.org/layout"
	"gioui.org/widget/material"
)

var blue = color.NRGBA{R: 0x40, G: 0x40, B: 0xC0, A: 0xFF}

type UserList struct {
	UserRegistry *types.Registry
}

func (l UserList) Layout(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	if l.UserRegistry != nil {
		userDetails := l.UserRegistry.GetUserDetails()
		usersList := layout.List{Axis: layout.Vertical}
		dim := usersList.Layout(gtx, len(userDetails), func(gtx layout.Context, index int) layout.Dimensions {
			return UserCard{DisplayType: "LAST_MESSAGE"}.Layout(gtx, userDetails[index], theme)
		})
		return dim
	}
	return layout.Dimensions{Size: gtx.Constraints.Min}

}
