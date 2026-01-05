// Copyright (C) 2026 Murilo Gomes Julio
// SPDX-License-Identifier: MIT

// Site: https://www.mugomes.com.br

package report

import (
	"fmt"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/jung-kurt/gofpdf"
)

const (
	A4WidthBase  = 595.0
	A4HeightBase = 842.0
	BaseFontSize = 14.0
)

// TEMA
type zoomTheme struct {
	fyne.Theme
	zoom float32
}

func (z *zoomTheme) Size(name fyne.ThemeSizeName) float32 {
	return z.Theme.Size(name) * z.zoom
}

func (z *zoomTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	return color.Black //z.Theme.Color(name, theme.VariantLight)
}

// COMPONENTE
type pageData struct {
	rect *canvas.Rectangle
	cont fyne.CanvasObject
}

type DocViewer struct {
	widget.BaseWidget
	content    *fyne.Container
	pages      []*pageData
	zoomFactor float32
}

func NewDocViewer() *DocViewer {
	dv := &DocViewer{content: container.NewVBox(), zoomFactor: 1.0}
	dv.ExtendBaseWidget(dv)
	return dv
}

func (dv *DocViewer) CreateRenderer() fyne.WidgetRenderer {
	centered := container.NewCenter(dv.content)
	scroll := container.NewScroll(centered)
	bg := canvas.NewRectangle(color.NRGBA{R: 45, G: 45, B: 48, A: 255})
	return widget.NewSimpleRenderer(container.NewStack(bg, scroll))
}

func (dv *DocViewer) AddPage(cont fyne.CanvasObject) {
	paper := canvas.NewRectangle(color.White)
	size := fyne.NewSize(A4WidthBase*dv.zoomFactor, A4HeightBase*dv.zoomFactor)
	paper.SetMinSize(size)
	paper.Resize(size)

	updateCanvasObjects(cont, dv.zoomFactor)
	themedContent := container.NewThemeOverride(cont, &zoomTheme{Theme: theme.DefaultTheme(), zoom: dv.zoomFactor})

	dv.pages = append(dv.pages, &pageData{rect: paper, cont: themedContent})
	dv.content.Add(container.NewPadded(container.NewStack(paper, container.NewPadded(themedContent))))
	dv.content.Refresh()
}

func (dv *DocViewer) SetZoom(factor float32) {
	if factor < 0.1 {
		factor = 0.1
	}
	dv.zoomFactor = factor
	newPointSize := fyne.NewSize(A4WidthBase*factor, A4HeightBase*factor)

	for _, p := range dv.pages {
		p.rect.SetMinSize(newPointSize)
		p.rect.Resize(newPointSize)
		if to, ok := p.cont.(*container.ThemeOverride); ok {
			to.Theme = &zoomTheme{Theme: theme.DefaultTheme(), zoom: factor}
			updateCanvasObjects(to.Content, factor)
			to.Refresh()
		}
	}
	dv.content.Refresh()
	dv.Refresh()
}

func updateCanvasObjects(obj fyne.CanvasObject, factor float32) {
	if obj == nil {
		return
	}
	if txt, ok := obj.(*canvas.Text); ok {
		txt.TextSize = BaseFontSize * factor
		txt.Refresh()
	}
	if c, ok := obj.(*fyne.Container); ok {
		for _, child := range c.Objects {
			updateCanvasObjects(child, factor)
		}
	}
}

func (dv *DocViewer) ExportToPDF(savePath string) error {
	pdf := gofpdf.New("P", "mm", "A4", "")
	tr := pdf.UnicodeTranslatorFromDescriptor("")

	// Proporção de escala: Fyne (595pt) para PDF (210mm)
	ratio := 210.0 / A4WidthBase

	for _, p := range dv.pages {
		pdf.AddPage()

		var drawAbsolute func(obj fyne.CanvasObject, offsetX, offsetY float32)
		drawAbsolute = func(obj fyne.CanvasObject, offsetX, offsetY float32) {
			if obj == nil || !obj.Visible() {
				return
			}

			if to, ok := obj.(*container.ThemeOverride); ok {
				drawAbsolute(to.Content, offsetX, offsetY)
				return
			}

			// Coordenadas Absolutas
			absX := float64(obj.Position().X+offsetX) * ratio
			absY := float64(obj.Position().Y+offsetY) * ratio
			objW := float64(obj.Size().Width) * ratio
			objH := float64(obj.Size().Height) * ratio

			// Margens do PDF (10mm)
			pdfX, pdfY := absX+10, absY+10

			switch realObj := obj.(type) {
			case *widget.Label:
				// Puxa o tamanho automaticamente
				// Se o label estiver num ThemeOverride, ele já terá o TextSize calculado
				// Caso contrário, usará a constante BaseFontSize ajustada pelo zoom global
				fSize := float64(BaseFontSize * dv.zoomFactor)

				pdf.SetFont("Arial", "", fSize)
				pdf.SetXY(pdfX, pdfY)
				pdf.MultiCell(objW, fSize*0.4, tr(realObj.Text), "", "L", false)

			case *canvas.Text:
				// Puxa o tamanho do objeto do Canvas
				fSize := float64(realObj.TextSize)

				pdf.SetFont("Arial", "", fSize)
				pdf.SetXY(pdfX, pdfY)

				// Calcula a altura baseada no fSize
				pdf.CellFormat(objW, fSize*0.5, tr(realObj.Text), "", 0, "L", false, 0, "")

			case *widget.Separator:
				middleY := pdfY + (objH / 2)
				pdf.SetDrawColor(180, 180, 180) // Cor cinza padrão
				pdf.SetLineWidth(0.2)
				pdf.Line(pdfX, middleY, pdfX+objW, middleY)
				pdf.SetDrawColor(0, 0, 0) // Volta para preto

			case *fyne.Container:
				for _, child := range realObj.Objects {
					drawAbsolute(child, obj.Position().X+offsetX, obj.Position().Y+offsetY)
				}
			}
		}

		drawAbsolute(p.cont, 0, 0)
	}

	return pdf.OutputFileAndClose(savePath)
}

func (dv *DocViewer) GetZoom() float32 { return dv.zoomFactor }

// API GLOBAL
var viewer = NewDocViewer()

func AddPage(cont fyne.CanvasObject) { viewer.AddPage(cont) }

func Preview(app fyne.App) {
	win := app.NewWindow("MGReport Preview")
	zoomLabel := widget.NewLabel("100%")

	toolbar := container.NewHBox(
		widget.NewLabelWithStyle("MGReport", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		layout.NewSpacer(),
		widget.NewButtonWithIcon("Exportar PDF", theme.DocumentSaveIcon(), func() {
			// USANDO URIWriteCloser para compatibilidade total com Fyne v2
			saveDlg := dialog.NewFileSave(func(writer fyne.URIWriteCloser, err error) {
				if err != nil {
					dialog.ShowError(err, win)
					return
				}
				if writer == nil {
					return
				}

				path := writer.URI().Path()
				writer.Close()

				err = viewer.ExportToPDF(path)
				if err != nil {
					dialog.ShowError(err, win)
				} else {
					dialog.ShowInformation("Sucesso", "PDF Salvo!", win)
				}
			}, win)
			saveDlg.SetFileName("relatorio.pdf")
			saveDlg.Show()
		}),
		widget.NewSeparator(),
		widget.NewButtonWithIcon("", theme.ZoomOutIcon(), func() {
			viewer.SetZoom(viewer.GetZoom() - 0.1)
			zoomLabel.SetText(fmt.Sprintf("%.0f%%", viewer.GetZoom()*100))
		}),
		zoomLabel,
		widget.NewButtonWithIcon("", theme.ZoomInIcon(), func() {
			viewer.SetZoom(viewer.GetZoom() + 0.1)
			zoomLabel.SetText(fmt.Sprintf("%.0f%%", viewer.GetZoom()*100))
		}),
	)

	win.SetContent(container.NewBorder(container.NewPadded(toolbar), nil, nil, nil, viewer))
	win.Resize(fyne.NewSize(1000, 800))
	win.Show()
}
