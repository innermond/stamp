package main

import (
	"flag"
	"fmt"
	"log"
	"path"

	"github.com/jung-kurt/gofpdf"
	"github.com/phpdave11/gofpdi"
	"github.com/pkg/errors"
	rscpdf "rsc.io/pdf"
)

var (
	fn, fout, stamp, postfix, unit string
	p, pos                         string
	xstamp, ystamp, wstamp         float64
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
	flag.Float64Var(&wstamp, "w", 30.0, "width stamp")

	flag.Parse()

	if fn == "" || stamp == "" {
		return errors.New("pdf file required")
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
		pg       rscpdf.Page
		pgw, pgh float64
	)
	// set default page media acordind to unit
	k := 1.0
	if unit == "mm" {
		k = 25.4 / 72
	}
	// assume document has same page dimensions expressed as points as page 1
	pg = rf.Page(1)
	pgw, pgh, err = getDimensions(pg, "MediaBox", k)
	if err != nil {
		err = errors.Wrap(err, "getDimensions")
		log.Fatal(err)
	}
	media := gofpdf.SizeType{pgw, pgh}
	pdf := gofpdf.NewCustom(&gofpdf.InitType{UnitStr: unit, Size: media})

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

	var fpdi = gofpdi.NewImporter()
	addtemplate, usetemplate := importer(pdf, fpdi)

	// add stamp template
	fpdi.SetSourceFile(stamp)
	box := "/MediaBox"
	stampid, err := addtemplate(1, box)
	if err != nil {
		err = errors.Wrap(err, "addtemplate")
		log.Fatal(err)
	}

	// we will import every pdf page from fn
	fpdi.SetSourceFile(fn)
	var (
		inx int
	)
	for i := 1; i <= np; i++ {
		if i > 1 {
			pg = rf.Page(i)
			pgwi, pghi, err := getDimensions(pg, "MediaBox", k)
			if err != nil {
				err = errors.Wrap(err, fmt.Sprintf("getDimensions page %d", i))
				log.Fatal(err)
			}
			if pgwi != pgw || pghi != pgh {
				pdf.SetPageBox("MediaBox", 0, 0, pgw, pgh)
			}
		}
		pdf.AddPage()
		// add page i as template
		tplid, err := addtemplate(i, box)
		if err != nil {
			err = errors.Wrap(err, fmt.Sprintf("addtemplate page %d", i))
			log.Fatal(err)
		}
		usetemplate(tplid, 0, 0, pgw, 0)
		if _, ok := selection[i]; ok {
			inx = whereIBelong(i, whenRangesEnd)
			xstamp = float64(positions[inx][0])
			ystamp = float64(positions[inx][1])
			pdf.SetAlpha(0.7, "Multiply")
			usetemplate(stampid, xstamp, ystamp, wstamp, 0.0)
			pdf.SetAlpha(1.0, "Normal")
		}
	}

	err = pdf.OutputFileAndClose(fout)
	if err != nil {
		err = errors.Wrap(err, "OutputFileAndClose")
		log.Fatal(err)
	}
}
