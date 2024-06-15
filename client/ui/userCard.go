package ui

import (
	"example/zerochat/client/types"
	"image"
	"image/color"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"golang.org/x/image/draw"
)

type UserCard struct {
	user    types.UserDetails
	message string
	btn     widget.Clickable
}

func (c *UserCard) Layout(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	border := widget.Border{Color: color.NRGBA{A: 0xff}, Width: unit.Dp(1)}
	return layout.Stack{}.Layout(
		gtx,
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
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
								if c.user.Avatar != nil {
									img := image.NewRGBA(image.Rectangle{Max: image.Point{X: dim, Y: dim}})
									draw.CatmullRom.Scale(img, img.Bounds(), c.user.Avatar, c.user.Avatar.Bounds(), draw.Src, nil)
									imgWidget := widget.Image{Src: paint.NewImageOp(img)}
									imgWidget.Scale = float32(dim) / float32(gtx.Dp(unit.Dp(float32(dim))))
									return imgWidget.Layout(gtx)
								} else {
									circle := clip.Ellipse{Max: image.Pt(dim, dim)}.Op(gtx.Ops)
									paint.FillShape(gtx.Ops, blue, circle)
									return layout.Dimensions{Size: image.Pt(dim, dim)}
								}
							})
							return dim
						}),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Vertical}.Layout(
								gtx,
								layout.Flexed(0.5, func(gtx layout.Context) layout.Dimensions {
									size := gtx.Constraints.Max
									label := material.Label(theme, unit.Sp(16), c.user.Name)
									label.MaxLines = 1
									label.Layout(gtx)
									return layout.Dimensions{Size: size}
								}),
								layout.Flexed(0.5, func(gtx layout.Context) layout.Dimensions {
									size := gtx.Constraints.Max
									label := material.Label(theme, unit.Sp(12), c.message)
									label.MaxLines = 1
									label.Layout(gtx)
									return layout.Dimensions{Size: size}
								}),
							)
						}),
					)
					return dim
				})
			})
		}),
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			return material.Clickable(gtx, &c.btn, func(gtx layout.Context) layout.Dimensions {
				return layout.Dimensions{Size: gtx.Constraints.Min}
			})
		}),
	)

}
