package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/tabwriter"
	"unicode/utf8"

	"github.com/gen2brain/iup-go/iup"
)

type Colors []string

var colors = Colors{
	"255 60 60",				// Red
	"60 200 60",				// Green
	"60 60 255",				// Blue

	// Red variations
	"230 200 60",				// Yellow
	"255 60 255",				// Pink

	// Green variations
	"60 220 220",				// Cyan
}

// Circular index
func (cls Colors) Elt(n int) string {
	return colors[n%len(cls)]
}

// MakeFieldsFunc returns a function that seprates on each character
// that is either a Space/Newline, Arabic (و) Waw character followed by space,
// or Aya seprator e.g. (13)
func MakeFieldsFunc() func(rune) bool {
	lastSeen := '.'
	sep := false				// ayat seperator

	isSpace := func (r rune) bool {
		if r == ' ' || r == '\n' || r == '\r' {
			return true
		}
		return false
	}
	
	f := func (r rune) bool {
		defer func () { lastSeen = r }()

		if r == '(' {
			sep = true
			return true
		}
		if r == ')' {
			sep = false
			return true
		}
		if sep {
			return true
		}

		if isSpace(r) || r == 'و' && isSpace(lastSeen) {
			return true
		}
		return false
	}
	return f
}

func Difference(positions map[string][]int) map[string][]int {
	m := make(map[string][]int)

	for k, s := range positions {
		m[k] = make([]int, len(s))
		m[k][0] = s[0]
		for i := range s {
			if i < 1 { continue }
			m[k][i] = s[i] - s[i-1] - 1
		}
	}

	return m
}

func Query(queries []string, fileName string) map[string][]int {
	sr := make(map[string][]int, len(queries))

	file, err := os.Open(fileName)
	defer file.Close()
	if err != nil { panic(err) }

	fileBytes, err := io.ReadAll(file)
	if err != nil { panic(err) }

	words := strings.FieldsFunc(string(fileBytes), MakeFieldsFunc())
	for i, w := range words {
		i++ 					// start from 1
		for _, q := range queries {
			if w == q {
				if _, ok := sr[w]; !ok {
					sr[w] = make([]int, 1, 5)
					sr[w][0] = i
				} else {
					sr[w] = append(sr[w], i)
				}
			}
		}
	}
	return Difference(sr)
}

func IndexAt(s, substr string, n int) int {
	i := strings.Index(s[n:], substr)
	if i == -1 {
		return -1
	}
	return i+n
}

func ColorWords(fileName string, queries []string, results map[string][]int, textField iup.Ihandle) {
	file, err := os.Open(fileName)
	if err != nil { panic(err) }
	defer file.Close()

	fileBytes, err := io.ReadAll(file)
	if err != nil { panic(err) }

	// IUP doesn't consider CR a character
	fileString := strings.ReplaceAll(string(fileBytes), "\r", "")
	textField.SetAttribute("VALUE", fileString)

	// Add statistics
	var stats strings.Builder
	FormatResults(results, &stats)
	textField.SetAttribute("APPEND", stats.String())

	for i, s := range queries {
		color := colors.Elt(i)
		re := regexp.MustCompile(fmt.Sprintf("%v[ \n]", s))
		indexes := re.FindAllStringIndex(fileString, -1)
		for _, v := range indexes {
			runeIndex := utf8.RuneCountInString(fileString[:v[0]])
			formatTag := iup.User()
			formatTag.SetAttributes(map[string]string{
				"FGCOLOR": "255 255 255",
				"BGCOLOR": color,
				"SELECTIONPOS": fmt.Sprintf("%v:%v", runeIndex, runeIndex+len([]rune(s))),
			})
			textField.SetAttribute("ADDFORMATTAG_HANDLE", formatTag)
		}
	}
}

func FormatResults(m map[string][]int, w io.Writer) {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)

	for _, k := range keys {
		fmt.Fprintf(tw, "%v\t", k)
		for _, p := range m[k] {
			fmt.Fprintf(tw, "%v\t", p)
		}
		fmt.Fprintln(tw)
	}

	tw.Flush()
}

func ChooseFile(isAllowNew bool) (string, error) {
	fileDlg := iup.FileDlg()
	defer fileDlg.Destroy()

	fileDlg.SetAttribute("TITLE", "Choose file:")
	if isAllowNew {
		fileDlg.SetAttribute("ALLOWNEW", "YES")
	} else {
		fileDlg.SetAttribute("ALLOWNEW", "NO")
	}

	iup.Popup(fileDlg, iup.CENTER, iup.CENTER)

	if fileDlg.GetInt("STATUS") < 0 {
		return "", fmt.Errorf("Please choose a file.")
	}

	return fileDlg.GetAttribute("VALUE"), nil
}

func main() {
	iup.Open()
	defer iup.Close()

	iup.SetGlobal("UTF8MODE", "YES")
	iup.SetGlobal("UTF8MODE_FILE", "YES")
	iup.SetGlobal("DEFAULTFONTSIZE", "12")

	var inputFile string
	var outputFile string

	inputButton := iup.Button("Choose input file...").SetAttributes(map[string]string{
		"PADDING": "5x5",
	})
	outputButton := iup.Button("Choose output file...").SetAttributes(map[string]string{
		"PADDING": "5x5",
	})
	inputButton.SetCallback("ACTION", iup.ActionFunc(func(ih iup.Ihandle) int {
		file, err := ChooseFile(false)
		if err == nil {
			inputFile = file
			ih.SetAttribute("TITLE", fmt.Sprintf("Input file: %v", filepath.Base(inputFile)))
			ih.SetAttribute("TIP", inputFile)
		}
		return iup.DEFAULT
	}))
	outputButton.SetCallback("ACTION", iup.ActionFunc(func(ih iup.Ihandle) int {
		file, err := ChooseFile(true)
		if err == nil {
			outputFile = file
			ih.SetAttribute("TITLE", fmt.Sprintf("Output file: %v", filepath.Base(outputFile)))
			ih.SetAttribute("TIP", outputFile)
		}
		return iup.DEFAULT
	}))

	hbox := iup.Hbox(inputButton, outputButton)

	showTextField := iup.Text().SetAttributes(map[string]string{
		"ALIGNMENT": "ACENTER",
		"EXPAND": "YES",
		"READONLY": "YES",
		"PADDING": "10x10",
		"MULTILINE": "YES",
		"WORDWRAP": "YES",
		"FORMATTING": "YES",
		"VISIBLELINES": "20",
		"VISIBLECOLUMNS": "30",
	})
	
	inputField := iup.Text().SetAttributes(map[string]string{
		"ALIGNMENT": "ARIGHT",
		"VISIBLECOLUMNS": "31",
	})

	runButton := iup.Button("Run").SetAttributes(map[string]string{
		"PADDING": "5x5",
	})
	runButton.SetCallback("ACTION", iup.ActionFunc(func(ih iup.Ihandle) int {
		if len(inputFile) == 0 && len(outputFile) == 0 {
			iup.Message("Error", "Please choose the input and output files.")
			return iup.ERROR
		} else {
			if len(inputFile) == 0 {
				iup.Message("Error", "Please choose the input file.")
				return iup.ERROR
			}
			if len(outputFile) == 0 {
				iup.Message("Error", "Please choose the output file.")
				return iup.ERROR
			}
		}

		queries := strings.Fields(inputField.GetAttribute("VALUE"))
		m := Query(queries, inputFile)
			
		ColorWords(inputFile, queries, m, showTextField)

		file, err := os.Create(outputFile)
		if err != nil { panic(err) }
		defer file.Close()
		FormatResults(m, file)

		iup.Message("Done.", "Operation complete.")
		
		return iup.DEFAULT
	}))

	vbox := iup.Vbox(showTextField, inputField, hbox, runButton).SetAttributes(map[string]string {
		"ALIGNMENT": "ACENTER",
		"GAP": "5",
		"MARGIN": "5x5",
	})

	dlg := iup.Dialog(vbox)
	dlg.SetAttributes(map[string]string{
		"TITLE": "Query words",
	})

	iup.Show(dlg)
	iup.MainLoop()
}
