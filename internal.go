package main

import (
	"errors"
	"log"
	"math"
	"strconv"
	"strings"

	"github.com/jung-kurt/gofpdf"
	"github.com/phpdave11/gofpdi"
	rscpdf "rsc.io/pdf"
)

func unpanic(err error) {
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
		gg := strings.Split(strings.TrimSpace(p), ",")
		var ll []string
		var a, z int
		for _, g := range gg {
			if strings.Contains(g, "-") {
				ll = strings.Split(g, "-")
			} else {
				ll = []string{g, g}
			}
			for i, l := range ll {
				ll[i] = strings.TrimSpace(l)
			}
			a, err = strconv.Atoi(ll[0])
			if err != nil {
				err = errors.New("wrong values for pages")
				return
			}
			z, err = strconv.Atoi(ll[1])
			if err != nil {
				err = errors.New("wrong values for pages")
				return
			}
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
	// be mercifull
	poss := strings.Split(strings.TrimSpace(pos), ",")
	var a, z int
	var pp []string
	for _, xy := range poss {
		pp = strings.Split(xy, "+")
		if len(pp) != 2 {
			err = errors.New("unexpected length position")
			return
		}
		for i, p := range pp {
			pp[i] = strings.TrimSpace(p)
		}
		a, err = strconv.Atoi(pp[0])
		if err != nil {
			err = errors.New("wrong values for position")
			return
		}
		z, err = strconv.Atoi(pp[1])
		if err != nil {
			err = errors.New("wrong values for position")
			return
		}
		positions = append(positions, []int{a, z})
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
