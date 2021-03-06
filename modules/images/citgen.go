package images

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"strings"
	"time"

	"github.com/Toffee-iZt/HwBot/bot"
	"github.com/Toffee-iZt/HwBot/bot/conversation"
	"github.com/Toffee-iZt/HwBot/vkapi"
	"github.com/Toffee-iZt/HwBot/vkapi/upload"
	"github.com/fogleman/gg"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
)

const copyrightSymbol = "©"

var citgen = bot.Command{
	Cmd:         []string{"citgen"},
	Description: "Генерация цитаты",
	Help:        "/citen и ответить или переслать сообщение",
	Options:     bot.OptionInChat | bot.OptionInDialog,
	Run: func(ctx *bot.Context, msg *bot.Message) {
		var fromID vkapi.ID
		var text string
		var t int64
		switch {
		case msg.Message.Reply != nil:
			r := msg.Message.Reply
			fromID, text = r.FromID, r.Text
			t = r.Date
		case len(msg.Message.Forward) > 0:
			f := msg.Message.Forward[0]
			fromID, text = f.FromID, f.Text
			t = f.Date
		default:
			ctx.ReplyText("Ответьте или перешлите сообщение")
		}
		if text == "" {
			ctx.ReplyText("Сообщение не содержит текста")
		}
		if len(msg.Args) > 0 {
			text = strings.Join(msg.Args, " ")
		}

		api := ctx.API()
		log := ctx.Log()

		name, photo, vkerr, err := getNamePhoto(api, fromID)
		if err != nil {
			log.Error("citgen: %s", err.Error())
			return
		}
		if vkerr != nil {
			log.Error("citgen: %s", vkerr.String())
			return
		}

		data, err := generateQuote(photo, name, text, time.Unix(t, 0), fromID == msg.Message.FromID, color.Black, color.White)
		if err != nil {
			log.Error("citgen generate: %s", err.Error())
			return
		}

		ctx.Reply(conversation.Message{
			Photos: []*conversation.Photo{conversation.NewPhoto("citgen.png", data)},
		})
	},
}

func getNamePhoto(api *vkapi.Client, from vkapi.ID) (string, image.Image, *vkapi.Error, error) {
	var name string
	var photo string
	if gid := from.ToGroup(); gid != 0 {
		g, err := api.GroupsGetByID(gid)
		if err != nil || len(g) == 0 {
			return "", nil, err, nil
		}
		name = g[0].Name
		photo = g[0].Photo200
	} else {
		u, err := api.UsersGet([]vkapi.UserID{from.ToUser()}, "", "photo_200")
		if err != nil || len(u) == 0 {
			return "", nil, err, nil
		}

		name = u[0].FirstName + " " + u[0].LastName
		photo = *u[0].Photo200
	}

	img, err := upload.DownloadImage(context.Background(), api, photo)
	if err != nil {
		return "", nil, nil, err
	}

	return name, img, nil, nil
}

const (
	fontSize = 20

	width     = 700
	minHeight = 400

	photoSize = 200

	textPointX = width / 3

	padding = 15
)

var fontFace = getGoFontFace()

func getGoFontFace() font.Face {
	gottf, _ := truetype.Parse(goregular.TTF)
	return truetype.NewFace(gottf, &truetype.Options{Size: fontSize})
}

func generateQuote(photo image.Image, name, quote string, t time.Time, self bool, bg, fg color.Color) (*bytes.Buffer, error) {
	lines, height := generateLines(quote, textPointX)

	name += " " + copyrightSymbol
	if self {
		name += " (self)"
	}

	dc := gg.NewContext(width, height)

	dc.SetFontFace(fontFace)
	dc.SetColor(bg)
	dc.Clear()
	dc.SetColor(fg)

	// Draw quote text
	for i, line := range lines {
		y := height/2 - fontSize*len(lines)/2 + i*fontSize
		dc.DrawString(line, textPointX, float64(y))
	}

	// Draw name
	dc.DrawString(name, float64(padding), float64(height-padding))

	// Draw time
	tstr := t.Format("02.01.2006 15:04")
	dc.DrawString(tstr, float64(width-getStringWidth(tstr)-padding), float64(height-padding))

	// Draw photo and make it round
	px, py := width/6, height/2
	dc.DrawEllipse(float64(px), float64(py), 100, 100)
	dc.Clip()
	dc.DrawImageAnchored(photo, px, py, 0.5, 0.5)

	data := bytes.NewBuffer(nil)
	return data, png.Encode(data, dc.Image())
}

func generateLines(s string, w int) ([]string, int) {
	var lines []string

	for _, line := range strings.Split(s, "\n") {
		var newLine string
		for _, word := range strings.Split(line, " ") {
			if getStringWidth(newLine+" "+word) > (width - w - 10) {
				lines = append(lines, newLine)
				newLine = word
			} else {
				newLine += " " + word
			}
		}

		if newLine != "" {
			lines = append(lines, strings.TrimSpace(newLine))
		}
	}

	lines[0] = "«" + lines[0]
	lines[len(lines)-1] += "»"

	h := len(lines) * (fontFace.Metrics().Height.Ceil() + 2)
	if h < minHeight {
		h = minHeight
	}

	return lines, h
}

func getStringWidth(s string) int {
	w := 0
	for _, r := range s {
		_, a, _ := fontFace.GlyphBounds(r)
		w += a.Round()
	}
	return w
}
