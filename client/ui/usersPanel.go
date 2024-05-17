package ui

import (
	"example/zerochat/client/types"
	"fmt"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget/material"
)

type UsersPanel struct {
	Self         types.UserDetails
	UserRegistry *types.Registry
}

func (up UsersPanel) Layout(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	fmt.Printf("Self is %v\n", up.Self)
	return layout.Flex{Axis: layout.Vertical}.Layout(
		gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Bottom: unit.Dp(10), Left: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				title := material.H6(theme, "You")
				return title.Layout(gtx)
			})
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Bottom: unit.Dp(30)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return UserCard{DisplayType: "SELF"}.Layout(gtx, up.Self, theme)
			})
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Bottom: unit.Dp(10), Left: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				title := material.H6(theme, "ONLINE USERS")
				return title.Layout(gtx)
			})
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return UserList{UserRegistry: up.UserRegistry}.Layout(gtx, theme)
		}),
	)
}
