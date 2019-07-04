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
	pg = rf.Page(1)
	box := "MediaBox"
	mbox := pg.V.Key(box)
	if mbox.Len() != 4 {
		log.Fatal(errors.New("mediabox wrong length"))
	}
	lx := mbox.Index(0).Float64()
	ly := mbox.Index(1).Float64()
	rx := mbox.Index(2).Float64()
	ry := mbox.Index(3).Float64()
	pgw, pgh = rx-lx, ry-ly

	// set default page media
	k := 1.0
	if unit == "mm" {
		k = 25.4 / 72
	}
	media := gofpdf.SizeType{pgw * k, pgh * k}
	pdf := gofpdf.NewCustom(&gofpdf.InitType{UnitStr: unit, Size: media})

	poss := strings.Split(pos, ",")
	var positions [][]int
	if len(poss) == 1 {
		pp := strings.Split(pos, "+")
		if len(pp) != 2 {
			log.Fatal(err)
		}
		a, err := strconv.Atoi(pp[0])
		z, err := strconv.Atoi(pp[1])
		if err != nil {
			log.Fatal(err)
		}
		positions = [][]int{[]int{a, z}}
	} else {
		var pp []string
		for _, xy := range poss {
			pp = strings.Split(xy, "+")
			if len(pp) != 2 {
				log.Fatal(err)
			}
			a, err := strconv.Atoi(pp[0])
			z, err := strconv.Atoi(pp[1])
			if err != nil {
				log.Fatal(err)
			}
			positions = append(positions, []int{a, z})
		}
	}

	var selection = map[int]bool{}
	var whenRangesEnd = []int{}

	if p != "" {
		gg := strings.Split(p, ",")
		var ll []string
		for _, g := range gg {
			if strings.Contains(g, "-") {
				ll = strings.Split(g, "-")
			} else {
				ll = []string{g, g}
			}
			a, err := strconv.Atoi(ll[0])
			z, err := strconv.Atoi(ll[1])
			if a > z {
				a, z = z, a
			}
			if err != nil {
				log.Fatal(err)
			}
			for inx := a; inx <= z; inx++ {
				selection[inx] = true
			}
			whenRangesEnd = append(whenRangesEnd, z)
		}
	} else {
		for i := 1; i <= np; i++ {
			selection[i] = true
		}
		whenRangesEnd = append(whenRangesEnd, np)
	}

	if len(positions) < len(whenRangesEnd) {
		whenRangesEnd = whenRangesEnd[:len(positions)]
	}

	var fpdi = gofpdi.NewImporter()
	fpdi.SetSourceFile(stamp)
	stampid := fpdi.ImportPage(1, "/"+box)
	// import template to page
	tplObjIDs := fpdi.PutFormXobjectsUnordered()
	pdf.ImportTemplates(tplObjIDs)
	imported := fpdi.GetImportedObjectsUnordered()
	pdf.ImportObjects(imported)
	importedObjPos := fpdi.GetImportedObjHashPos()
	pdf.ImportObjPos(importedObjPos)

	fpdi.SetSourceFile(fn)

	var (
		inx int
	)
	for i := 1; i <= np; i++ {
		pdf.AddPage()

		tplid := fpdi.ImportPage(i, "/"+box)
		// import template to page
		tplObjIDs := fpdi.PutFormXobjectsUnordered()
		pdf.ImportTemplates(tplObjIDs)
		imported := fpdi.GetImportedObjectsUnordered()
		pdf.ImportObjects(imported)
		importedObjPos := fpdi.GetImportedObjHashPos()
		pdf.ImportObjPos(importedObjPos)
		tplname, sx, sy, tx, ty := fpdi.UseTemplate(tplid, 0, 0, pgw*k, 0)
		pdf.UseImportedTemplate(tplname, sx, sy, tx, ty)
		if _, ok := selection[i]; ok {
			inx = whereIBelong(i, whenRangesEnd)
			xstamp = float64(positions[inx][0])
			ystamp = float64(positions[inx][1])
			pdf.SetAlpha(0.7, "Multiply")
			stamptpl, sx, sy, tx, ty := fpdi.UseTemplate(stampid, xstamp, ystamp, wstamp, 0.0)
			pdf.UseImportedTemplate(stamptpl, sx, sy, tx, ty)
			pdf.SetAlpha(1.0, "Normal")
		}
	}

	err = pdf.OutputFileAndClose(fout)
	if err != nil {
		log.Fatal(err)
	}
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
