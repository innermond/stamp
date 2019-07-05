package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"math"
	"path"
	"strconv"
	"strings"

	"github.com/jung-kurt/gofpdf"
	"github.com/phpdave11/gofpdi"
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
		log.Fatal(err)
	}
	media := gofpdf.SizeType{pgw, pgh}
	pdf := gofpdf.NewCustom(&gofpdf.InitType{UnitStr: unit, Size: media})

	positions, err := positionsFromInput(pos)
	if err != nil {
		log.Fatal(err)
	}

	selection, whenRangesEnd, err := pagesFromInput(p, np)
	if err != nil {
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
		log.Fatal(err)
	}

	// we will import every pdf page from fn
	fpdi.SetSourceFile(fn)
	var (
		inx int
	)
	for i := 1; i <= np; i++ {
		pdf.AddPage()
		// add page i as template
		tplid, err := addtemplate(i, box)
		if err != nil {
			log.Fatal(err)
		}
		//
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
		log.Fatal(err)
	}
}

var unpanic = func(err error) {
	if r := recover(); r != nil {
		var ok bool
		err, ok = r.(error)
		log.Println(err)
		if !ok {
			err = errors.New("expected to get an error from recover")
			log.Println(err)
		}
	}
}

// pagesFromInput process a string that represents pages chosen to be stamped
// returns 2 slices
// selection is a map where pages selected are there as [int]bool{1:true,10:true}
// whenRangesEnd contain ends of ranges. 2-3 has 3 as end, 200-700 has 700 as end
// ex: p is '1,10,45-5,17-16' results in a whenRangesEnd []int{1,10,45,17}
func pagesFromInput(p string, np int) (selection map[int]bool, whenRangesEnd []int, err error) {
	selection = map[int]bool{}
	whenRangesEnd = []int{}

	if p != "" {
		gg := strings.Split(p, ",")
		var ll []string
		var a, z int
		for _, g := range gg {
			if strings.Contains(g, "-") {
				ll = strings.Split(g, "-")
			} else {
				ll = []string{g, g}
			}
			a, err = strconv.Atoi(ll[0])
			z, err = strconv.Atoi(ll[1])
			if a > z {
				a, z = z, a
			}
			if err != nil {
				err = errors.New("wrong values for pages")
				return
			}
			for inx := a; inx <= z; inx++ {
				selection[inx] = true
			}
			whenRangesEnd = append(whenRangesEnd, z)
		}
	} else {
		// add all pages
		for i := 1; i <= np; i++ {
			selection[i] = true
		}
		// limits are start and end page
		whenRangesEnd = append(whenRangesEnd, np)
	}

	return
}

func positionsFromInput(pos string) (positions [][]int, err error) {
	poss := strings.Split(pos, ",")
	var a, z int
	if len(poss) == 1 {
		pp := strings.Split(pos, "+")
		if len(pp) != 2 {
			err = errors.New("unexpected length position")
			return
		}
		a, err = strconv.Atoi(pp[0])
		z, err = strconv.Atoi(pp[1])
		if err != nil {
			err = errors.New("wrong values for position")
			return
		}
		positions = [][]int{[]int{a, z}}
	} else {
		var pp []string
		for _, xy := range poss {
			pp = strings.Split(xy, "+")
			if len(pp) != 2 {
				err = errors.New("unexpected length position")
				return
			}
			a, err = strconv.Atoi(pp[0])
			z, err = strconv.Atoi(pp[1])
			if err != nil {
				err = errors.New("wrong values for position")
				return
			}
			positions = append(positions, []int{a, z})
		}
	}
	return
}

// importer returns 2 funcs that have access to common resources gofpdf.Fpdf and gofpdi.importer(
// it creates a common context
func importer(pdf *gofpdf.Fpdf, fpdi *gofpdi.Importer) (
	func(i int, box string) (int, error),
	func(int, float64, float64, float64, float64) error,
) {
	return func(i int, box string) (tplid int, err error) {
			defer unpanic(err)
			box = strings.TrimLeft(box, "/")
			tplid = fpdi.ImportPage(i, "/"+box)
			// import template to page
			tplObjIDs := fpdi.PutFormXobjectsUnordered()
			pdf.ImportTemplates(tplObjIDs)
			imported := fpdi.GetImportedObjectsUnordered()
			pdf.ImportObjects(imported)
			importedObjPos := fpdi.GetImportedObjHashPos()
			pdf.ImportObjPos(importedObjPos)
			return tplid, nil
		}, func(tplid int, x, y, w, h float64) (err error) {
			defer unpanic(err)
			tplname, sx, sy, tx, ty := fpdi.UseTemplate(tplid, x, y, w, h)
			pdf.UseImportedTemplate(tplname, sx, sy, tx, ty)
			return
		}
}

func getDimensions(pg rscpdf.Page, box string, k float64) (width float64, height float64, err error) {
	mbox := pg.V.Key(box)
	if mbox.Len() != 4 {
		err = errors.New("mediabox wrong length")
		return
	}

	lx := mbox.Index(0).Float64()
	ly := mbox.Index(1).Float64()
	rx := mbox.Index(2).Float64()
	ry := mbox.Index(3).Float64()

	return (rx - lx) * k, (ry - ly) * k, nil
}

func whereIBelong(i int, limits []int) int {

	var (
		inx      int
		x, xpeak int
		found    = math.MaxInt64
	)

	for x, xpeak = range limits {
		if i <= xpeak {
			if found > xpeak {
				found = xpeak
				inx = x
				continue
			}
		}
	}

	return inx
}
