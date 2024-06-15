package ui

import (
	"bufio"
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"os"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"gioui.org/x/component"
	"gioui.org/x/explorer"
	"golang.org/x/image/draw"
)

const IMAGE_SIZE = 70

type imageResult struct {
	imageData image.Image
	err       error
}

type ProfilePanel struct {
	input       component.TextField
	button      widget.Clickable
	photobutton widget.Clickable
	expl        explorer.Explorer
	imgChan     chan imageResult
	Avatar      []byte
	OnConfirm   func(string)
	OnImageLoad func([]byte)
}

func CreateDefaultImage() ([]byte, error) {
	data, err := os.ReadFile("placeholder.png")
	if err != nil {
		fmt.Printf("Error %s\n", err)
		return nil, err
	}
	reader := bytes.NewReader([]byte(data))
	img, _, err := image.Decode(reader)
	if err != nil {
		fmt.Printf("Error decoding placeholder %s\n", err)
		return nil, err
	}

	scaledImg := image.NewRGBA(image.Rectangle{Max: image.Point{X: IMAGE_SIZE, Y: IMAGE_SIZE}})
	draw.CatmullRom.Scale(scaledImg, scaledImg.Bounds(), img, img.Bounds(), draw.Src, nil)

	var b bytes.Buffer
	w := bufio.NewWriter(&b)
	jpeg.Encode(w, scaledImg, nil)
	return b.Bytes(), nil
}

func (profile *ProfilePanel) processEvents(gtx layout.Context) {
	if profile.imgChan == nil {
		profile.imgChan = make(chan imageResult)
	}
	if profile.button.Clicked(gtx) && profile.OnConfirm != nil {
		if profile.input.Text() != "" {
			profile.OnConfirm(profile.input.Text())
		} else {
			profile.input.SetError("Empty nickname")
		}
	}
	if profile.photobutton.Clicked(gtx) {
		go func() {
			file, err := profile.expl.ChooseFile("png", "jpeg", "jpg")
			if err != nil {
				fmt.Println(err)
				profile.imgChan <- imageResult{err: err}
				return
			}
			defer file.Close()
			imgData, _, err := image.Decode(file)
			if err != nil {
				fmt.Println(err)
				profile.imgChan <- imageResult{err: err}
				return
			}
			profile.imgChan <- imageResult{imageData: imgData}
		}()

		go func() {
			img := <-profile.imgChan
			fmt.Println("channel received image")
			if img.err != nil {
				fmt.Println(img.err)
				return
			}
			w := img.imageData.Bounds().Dx()
			h := img.imageData.Bounds().Dy()
			var subimg image.Image
			if w >= h {
				dx := (w - h) / 2
				subimg = img.imageData.(interface {
					SubImage(r image.Rectangle) image.Image
				}).SubImage(image.Rect(dx, 0, h, h))
			} else if w < h {
				dy := (h - w) / 2
				subimg = img.imageData.(interface {
					SubImage(r image.Rectangle) image.Image
				}).SubImage(image.Rect(0, dy, w, w))
			}

			scaledImg := image.NewRGBA(image.Rectangle{Max: image.Point{X: IMAGE_SIZE, Y: IMAGE_SIZE}})
			draw.CatmullRom.Scale(scaledImg, scaledImg.Bounds(), subimg, subimg.Bounds(), draw.Src, nil)

			var b bytes.Buffer
			writer := bufio.NewWriter(&b)
			jpeg.Encode(writer, scaledImg, nil)
			imgBytes := b.Bytes()

			profile.Avatar = imgBytes
			profile.OnImageLoad(imgBytes)
		}()
	}
}

func (profile *ProfilePanel) Layout(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	profile.processEvents(gtx)
	return layout.Flex{Axis: layout.Vertical, Spacing: layout.SpaceAround, Alignment: layout.Middle}.Layout(
		gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Max.X = 700
			return layout.Flex{}.Layout(
				gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Stack{Alignment: layout.Center}.Layout(
						gtx,
						layout.Expanded(func(gtx layout.Context) layout.Dimensions {
							return material.Clickable(gtx, &profile.photobutton, func(gtx layout.Context) layout.Dimensions {
								return layout.Dimensions{Size: gtx.Constraints.Min}
							})
						}),
						layout.Stacked(func(gtx layout.Context) layout.Dimensions {
							dim := layout.UniformInset(unit.Dp(5)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								dim := 70
								gtx.Constraints.Max = image.Point{X: dim, Y: dim}
								if profile.Avatar != nil {
									decoded, _, err := image.Decode(bytes.NewReader(profile.Avatar))
									if err != nil {
										circle := clip.Ellipse{Max: image.Pt(dim, dim)}.Op(gtx.Ops)
										paint.FillShape(gtx.Ops, blue, circle)
										return layout.Dimensions{Size: image.Pt(dim, dim)}
									}
									imgWidget := widget.Image{Src: paint.NewImageOp(decoded)}
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
					)
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(20)}.Layout),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					profile.input.CharLimit = 20
					profile.input.MaxLen = 20
					profile.input.Filter = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz_"
					return profile.input.Layout(gtx, theme, "Enter a nickname")
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(20)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Top: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return material.Button(theme, &profile.button, "Confirm").Layout(gtx)
					})
				}),
			)
		}),
	)
}
