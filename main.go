package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"strings"

	"github.com/jung-kurt/gofpdf"
	pdi "github.com/jung-kurt/gofpdf/contrib/gofpdi"
	"github.com/pkg/errors"
	rscpdf "rsc.io/pdf"
)

var (
	fn, fout, stamp, postfix, unit string
	p, pos                         string
	xstamp, ystamp, wstamp, hstamp float64
	alpha                          float64
	overcolor                      string
)

func initFlags() error {

	var err error

	flag.StringVar(&fn, "f", "", "pdf file to be stamped")
	flag.StringVar(&fout, "o", "", "stamped pdf file")
	flag.StringVar(&stamp, "s", "", "stamp file")
	flag.StringVar(&postfix, "postfix", "stamped", "termination added to stamped filename")
	flag.StringVar(&unit, "unit", "mm", "unit of measurements")
	flag.StringVar(&p, "p", "", "pages that are to be stamped")
	flag.StringVar(&pos, "pos", "", "stamp position as x+y. ex: 400+500")
	flag.Float64Var(&xstamp, "x", 0.0, "xpos stamp")
	flag.Float64Var(&ystamp, "y", 0.0, "ypos stamp")
	flag.Float64Var(&wstamp, "w", 0.0, "width stamp")
	flag.Float64Var(&hstamp, "h", 0.0, "height stamp")
	flag.Float64Var(&alpha, "alpha", 0.7, "transparency from 0.0 invisible to 1.0 opaque")
	flag.StringVar(&overcolor, "overcolor", "Multiply", "how stamo color will blend with under colors")

	flag.Parse()

	if fn == "" || stamp == "" {
		return errors.New("pdf file required")
	}
	ext := strings.ToLower(path.Ext(stamp))
	var ok bool
	switch ext {
	case ".pdf", ".png", ".jpg", ".jpeg":
		ok = true
	}
	if !ok {
		return errors.New("stamp type unacceptable")
	}

	if fout == "" {
		ext := path.Ext(fn)
		fout = fmt.Sprintf("%s.%s%s", fn[:len(fn)-len(ext)], postfix, ext)
	}

	return err
}

func main() {

	err := initFlags()
	if err != nil {
		log.Fatal(err)
	}

	// get num pages and box size
	rf, err := rscpdf.Open(fn)
	if err != nil {
		log.Fatal(err)
	}

	np := rf.NumPage()
	var (
		pg         rscpdf.Page
		pgw, pgh   float64
		box        = "/MediaBox"
		boxNoSlash = strings.TrimLeft(box, "/")
	)
	// set default page media acordind to unit
	k := 1.0
	if unit == "mm" {
		k = 25.4 / 72
	}
	// assume document has same page dimensions expressed as points as page 1
	pg = rf.Page(1)
	pgw, pgh, err = getDimensions(pg, boxNoSlash, k)
	if err != nil {
		err = errors.Wrap(err, "getDimensions")
		log.Fatal(err)
	}
	media := gofpdf.SizeType{pgw, pgh}
	pdf := gofpdf.NewCustom(&gofpdf.InitType{UnitStr: unit, Size: media})
	wout, _ := os.Create(fout)
	pdf.FlushTo(wout)
	pdf.PutHeader()

	positions, err := positionsFromInput(pos)
	if err != nil {
		err = errors.Wrap(err, "positionsFromInput")
		log.Fatal(err)
	}

	selection, whenRangesEnd, err := pagesFromInput(p, np)
	if err != nil {
		err = errors.Wrap(err, "pagesFromInput")
		log.Fatal(err)
	}

	if len(positions) < len(whenRangesEnd) {
		whenRangesEnd = whenRangesEnd[:len(positions)]
	}

	var (
		stampid      int
		stampIsImage bool
		ext          string
		opt          = gofpdf.ImageOptions{}
	)
	if ext = path.Ext(stamp)[1:]; ext != "pdf" {
		stampIsImage = true

		fimg, err := os.Open(stamp)
		if err != nil {
			log.Fatal(err)
		}
		defer fimg.Close()

		opt = gofpdf.ImageOptions{
			ReadDpi:   true,
			ImageType: ext,
		}
		pdf.RegisterImageOptionsReader(stamp, opt, fimg)
	} else {
		stampid = pdi.ImportPage(pdf, stamp, 1, box)
	}

	var (
		inx int
	)
	for i := 1; i <= np; i++ {
		if i > 1 {
			pg = rf.Page(i)
			pgw, pgh, err = getDimensions(pg, boxNoSlash, k)
			if err != nil {
				err = errors.Wrap(err, fmt.Sprintf("getDimensions page %d", i))
				log.Fatal(err)
			}
			pdf.SetPageBox(boxNoSlash, 0, 0, pgw, pgh)
		}
		pdf.AddPage()
		// add page i as template
		tplid := pdi.ImportPage(pdf, fn, i, box)
		pdi.UseImportedTemplate(pdf, tplid, 0, 0, pgw, pgh)
		if _, ok := selection[i]; ok {
			inx = whereIBelong(i, whenRangesEnd)
			xstamp = float64(positions[inx][0])
			ystamp = float64(positions[inx][1])
			pdf.SetAlpha(alpha, overcolor)
			if stampIsImage {
				pdf.ImageOptions(stamp, xstamp, ystamp, wstamp, hstamp, false, opt, 0, "")
			} else {
				pdi.UseImportedTemplate(pdf, stampid, xstamp, ystamp, wstamp, hstamp)
			}
			pdf.SetAlpha(1.0, "Normal")
		}
		pdf.WritePage()
	}

	pdf.Close()
	err = wout.Close()
	if err != nil {
		err = errors.Wrap(err, "OutputFileAndClose")
		log.Fatal(err)
	}
}
