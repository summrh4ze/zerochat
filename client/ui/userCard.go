package ui

import (
	"example/zerochat/client/types"
	"fmt"
	"image"
	"image/color"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

type UserCard struct {
	UserRegistry *types.Registry
	DisplayType  string
}

func (c UserCard) Layout(gtx layout.Context, user types.UserDetails, theme *material.Theme) layout.Dimensions {
	border := widget.Border{Color: color.NRGBA{A: 0xff}, Width: unit.Dp(1)}
	return layout.Inset{Bottom: unit.Dp(2)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return border.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Max.X = 300
			gtx.Constraints.Max.Y = 60
			dim := layout.Flex{Alignment: layout.Middle, Spacing: layout.SpaceBetween}.Layout(
				gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					dim := layout.UniformInset(unit.Dp(5)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						dim := gtx.Constraints.Max.Y
						gtx.Constraints.Max = image.Point{X: dim, Y: dim}
						circle := clip.Ellipse{Max: image.Pt(dim, dim)}.Op(gtx.Ops)
						paint.FillShape(gtx.Ops, blue, circle)
						return layout.Dimensions{Size: image.Pt(dim, dim)}
					})
					return dim
				}),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					fmt.Printf("INSIDE tab label constraints = %#v\n", gtx.Constraints)
					return layout.Flex{Axis: layout.Vertical}.Layout(
						gtx,
						layout.Flexed(0.5, func(gtx layout.Context) layout.Dimensions {
							size := gtx.Constraints.Max
							label := material.Label(theme, unit.Sp(16), user.Name)
							label.MaxLines = 1
							label.Layout(gtx)
							return layout.Dimensions{Size: size}
						}),
						layout.Flexed(0.5, func(gtx layout.Context) layout.Dimensions {
							size := gtx.Constraints.Max
							txt := ""
							switch c.DisplayType {
							case "SELF":
								txt = "This is your profile"
							case "LAST_MESSAGE":
								txt = "Say Hi!"
							}
							label := material.Label(theme, unit.Sp(12), txt)
							label.MaxLines = 1
							label.Layout(gtx)
							return layout.Dimensions{Size: size}
						}),
					)
				}),
			)
			fmt.Printf("SIZE OF TAB FLEX = %#v\n", dim)
			return dim
		})
	})
}
