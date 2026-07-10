package cmd

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	textm "github.com/yuin/goldmark/text"
)

const (
	codeFrameOverhead = 5 // selection rail plus the frame's content prefix

	ansiReset    = "\x1b[0m"
	ansiBold     = "\x1b[1m"
	ansiRed      = "\x1b[31m"
	ansiGreen    = "\x1b[32m"
	ansiYellow   = "\x1b[33m"
	ansiBlue     = "\x1b[34m"
	ansiCyan     = "\x1b[36m"
	ansiBoldBlue = "\x1b[1;34m"
)

type renderedDocument struct {
	text       string
	blockLines []int
}

type headingContext struct {
	id     int
	level  int
	text   string
	prefix string
}

type terminalRenderer struct {
	source     []byte
	width      int
	color      bool
	focused    int
	running    int
	cancelling int
	results    map[int]executionResult
	lines      []string
	blockLines []int
	executable int
	headings   [6]headingContext
	emitted    [6]headingContext
	headingID  int
	compact    bool
}

func renderMarkdown(source []byte, width int, color bool, focused, running int, results map[int]executionResult) renderedDocument {
	return renderMarkdownMode(source, width, color, focused, running, results, false)
}

func renderMarkdownMode(source []byte, width int, color bool, focused, running int, results map[int]executionResult, compact bool) renderedDocument {
	return renderReaderMarkdown(source, width, color, focused, running, results, compact, -1)
}

func renderReaderMarkdown(source []byte, width int, color bool, focused, running int, results map[int]executionResult, compact bool, cancelling int) renderedDocument {
	markdown := goldmark.New(goldmark.WithExtensions(extension.GFM))
	document := markdown.Parser().Parse(textm.NewReader(source))
	renderer := &terminalRenderer{source: source, width: max(20, width), color: color, focused: focused, running: running, cancelling: cancelling, results: results, compact: compact}
	for node := document.FirstChild(); node != nil; node = node.NextSibling() {
		renderer.renderBlock(node, "")
	}
	return renderedDocument{text: strings.Join(renderer.lines, "\n"), blockLines: renderer.blockLines}
}

func (r *terminalRenderer) renderBlock(node ast.Node, prefix string) {
	switch current := node.(type) {
	case *ast.Heading:
		r.headingID++
		heading := headingContext{id: r.headingID, level: current.Level, text: cleanText(r.inlineText(current)), prefix: prefix}
		r.headings[current.Level-1] = heading
		for level := current.Level; level < len(r.headings); level++ {
			r.headings[level] = headingContext{}
		}
		if r.compact {
			return
		}
		r.renderHeading(heading)
	case *ast.Paragraph, *ast.TextBlock:
		if r.compact {
			return
		}
		r.lines = append(r.lines, wrapText(cleanText(r.inlineText(node)), r.width-width(prefix), prefix)...)
		r.blank()
	case *ast.FencedCodeBlock:
		r.renderFence(current, prefix)
	case *ast.CodeBlock:
		if r.compact {
			return
		}
		r.renderPlainCode(string(current.Lines().Value(r.source)), prefix)
	case *ast.Blockquote:
		for child := current.FirstChild(); child != nil; child = child.NextSibling() {
			r.renderBlock(child, prefix+"▌ ")
		}
	case *ast.List:
		if r.compact {
			return
		}
		r.renderList(current, prefix)
	case *ast.ThematicBreak:
		if r.compact {
			return
		}
		r.lines = append(r.lines, strings.Repeat("─", max(3, r.width)))
		r.blank()
	default:
		if r.compact {
			return
		}
		if node.Kind().String() == "Table" {
			r.renderTable(node, prefix)
			return
		}
		text := cleanText(string(node.Text(r.source)))
		if text != "" {
			r.lines = append(r.lines, wrapText(text, r.width-width(prefix), prefix)...)
			r.blank()
		}
	}
}

func (r *terminalRenderer) renderFence(node *ast.FencedCodeBlock, prefix string) {
	info := ""
	if node.Info != nil {
		info = string(node.Info.Text(r.source))
	}
	language, label, executable := fenceMetadata(info)
	code := strings.TrimSuffix(string(node.Lines().Value(r.source)), "\n")
	if !executable {
		if r.compact {
			return
		}
		r.renderCodeFrame(-1, prefix, label, language, code, "display only", "", executionResult{})
		return
	}
	if r.compact {
		r.renderCompactHeadings()
	}

	index := r.executable
	r.executable++
	result, executed := r.results[index]
	status, style := "ready", ansiCyan
	if index == r.cancelling {
		status, style = "cancelling", ansiYellow
	} else if index == r.running {
		status, style = "running", ansiBlue
	} else if executed && result.cancelled {
		status, style = "cancelled", ansiYellow
	} else if executed && result.exitCode == 0 {
		status, style = "success", ansiGreen
	} else if executed {
		status, style = fmt.Sprintf("failed (exit %d)", result.exitCode), ansiRed
	}
	if !executed && index != r.running {
		if index == r.focused {
			status = "ready to execute"
		} else {
			status = "executable"
		}
	}
	r.renderCodeFrame(index, prefix, label, language, code, status, style, result)
}

func (r *terminalRenderer) renderHeading(heading headingContext) {
	r.blank()
	switch heading.level {
	case 1:
		r.lines = append(r.lines, heading.prefix+r.paint(heading.text, ansiBold))
		r.lines = append(r.lines, heading.prefix+strings.Repeat("═", max(3, r.width-width(heading.prefix))))
	case 2:
		r.lines = append(r.lines, heading.prefix+r.paint(heading.text, ansiBold))
		r.lines = append(r.lines, heading.prefix+strings.Repeat("─", max(3, r.width-width(heading.prefix))))
	default:
		caption := strings.Repeat("#", heading.level) + " " + heading.text
		r.lines = append(r.lines, heading.prefix+r.paint(caption, ansiBold))
	}
	r.blank()
}

func (r *terminalRenderer) renderCompactHeadings() {
	firstChanged := -1
	for level := range r.headings {
		if r.headings[level].id != r.emitted[level].id {
			firstChanged = level
			break
		}
	}
	if firstChanged < 0 {
		return
	}
	for level := firstChanged; level < len(r.headings); level++ {
		if r.headings[level].id != 0 {
			r.renderHeading(r.headings[level])
		}
	}
	r.emitted = r.headings
}

func (r *terminalRenderer) renderPlainCode(code, prefix string) {
	for _, line := range strings.Split(strings.TrimSuffix(code, "\n"), "\n") {
		r.lines = append(r.lines, prefix+line)
	}
	r.blank()
}

func (r *terminalRenderer) renderCodeFrame(index int, prefix, label, language, code, status, style string, result executionResult) {
	frameWidth := max(20, r.width-width(prefix)-(codeFrameOverhead-2))
	if index >= 0 {
		r.blockLines = append(r.blockLines, len(r.lines))
	}
	r.frameLine(index, prefix, r.borderLine("┌─", label, language, frameWidth, style, true, false))
	for _, sourceLine := range strings.Split(code, "\n") {
		for _, line := range wrapPreserved(sourceLine, frameWidth-2) {
			r.frameLine(index, prefix, r.paint("│ ", style)+line)
		}
	}
	if result.stdout != "" {
		r.frameLine(index, prefix, r.borderLine("├─", "stdout", "", frameWidth, style, true, false))
		for _, sourceLine := range strings.Split(strings.TrimSuffix(result.stdout, "\n"), "\n") {
			for _, line := range wrapPreserved(sourceLine, frameWidth-2) {
				r.frameLine(index, prefix, r.paint("│ ", style)+line)
			}
		}
	}
	if result.stderr != "" {
		r.frameLine(index, prefix, r.borderLine("├─", "stderr", "", frameWidth, style, true, false))
		for _, sourceLine := range strings.Split(strings.TrimSuffix(result.stderr, "\n"), "\n") {
			for _, line := range wrapPreserved(sourceLine, frameWidth-2) {
				r.frameLine(index, prefix, r.paint("│ ", style)+line)
			}
		}
	}
	r.frameLine(index, prefix, r.borderLine("└─", "", status, frameWidth, style, false, true))
	r.blank()
}

func (r *terminalRenderer) frameLine(index int, prefix, content string) {
	rail := "   "
	if index >= 0 && index == r.focused {
		rail = r.paint("▌", ansiBoldBlue) + "  "
	}
	r.lines = append(r.lines, rail+prefix+content)
}

func (r *terminalRenderer) borderLine(left, label, right string, lineWidth int, style string, labelBold, rightBold bool) string {
	leftLabel := ""
	if label != "" {
		leftLabel = " " + label + " "
	}
	rightLabel := ""
	if right != "" {
		rightLabel = " " + right + " "
	}
	filler := strings.Repeat("─", max(1, lineWidth-width(left)-width(leftLabel)-width(rightLabel)-1))
	result := r.paint(left, style)
	labelStyle := style
	if labelBold {
		labelStyle = ansiBold + style
	}
	result += r.paint(leftLabel, labelStyle) + r.paint(filler, style)
	rightStyle := style
	if rightBold {
		rightStyle = ansiBold + style
	}
	result += r.paint(rightLabel, rightStyle) + r.paint("─", style)
	return result
}

func (r *terminalRenderer) renderList(list *ast.List, prefix string) {
	number := list.Start
	for item := list.FirstChild(); item != nil; item = item.NextSibling() {
		marker := "• "
		if list.IsOrdered() {
			marker = fmt.Sprintf("%d. ", number)
			number++
		}
		text := cleanText(string(item.Text(r.source)))
		wrapped := wrapText(text, r.width-width(prefix+marker), "")
		for index, line := range wrapped {
			if index == 0 {
				r.lines = append(r.lines, prefix+marker+line)
			} else {
				r.lines = append(r.lines, prefix+strings.Repeat(" ", width(marker))+line)
			}
		}
	}
	r.blank()
}

func (r *terminalRenderer) renderTable(table ast.Node, prefix string) {
	var rows [][]string
	for row := table.FirstChild(); row != nil; row = row.NextSibling() {
		var cells []string
		for cell := row.FirstChild(); cell != nil; cell = cell.NextSibling() {
			cells = append(cells, cleanText(string(cell.Text(r.source))))
		}
		rows = append(rows, cells)
	}
	widths := make([]int, 0)
	for _, row := range rows {
		for column, cell := range row {
			for len(widths) <= column {
				widths = append(widths, 0)
			}
			widths[column] = max(widths[column], width(cell))
		}
	}
	for rowIndex, row := range rows {
		var line strings.Builder
		line.WriteString(prefix)
		for column, cell := range row {
			line.WriteString(cell)
			if column < len(row)-1 {
				line.WriteString(strings.Repeat(" ", widths[column]-width(cell)+2))
			}
		}
		value := line.String()
		if rowIndex == 0 {
			value = r.paint(value, ansiBold)
		}
		r.lines = append(r.lines, value)
	}
	r.blank()
}

func (r *terminalRenderer) paint(text, style string) string {
	if !r.color || style == "" || text == "" {
		return text
	}
	return style + text + ansiReset
}

func (r *terminalRenderer) inlineText(node ast.Node) string {
	var result strings.Builder
	var visit func(ast.Node)
	visit = func(current ast.Node) {
		switch value := current.(type) {
		case *ast.Text:
			result.Write(value.Value(r.source))
			if value.SoftLineBreak() {
				result.WriteByte(' ')
			}
			if value.HardLineBreak() {
				result.WriteByte('\n')
			}
			return
		case *ast.String:
			result.Write(value.Value)
			return
		case *ast.Emphasis:
			if r.color {
				result.WriteString(ansiBold)
			}
			for child := current.FirstChild(); child != nil; child = child.NextSibling() {
				visit(child)
			}
			if r.color {
				result.WriteString(ansiReset)
			}
			return
		}
		for child := current.FirstChild(); child != nil; child = child.NextSibling() {
			visit(child)
		}
	}
	visit(node)
	return result.String()
}

func (r *terminalRenderer) blank() {
	if len(r.lines) == 0 || r.lines[len(r.lines)-1] != "" {
		r.lines = append(r.lines, "")
	}
}

func cleanText(value string) string { return strings.Join(strings.Fields(value), " ") }

func wrapText(value string, limit int, prefix string) []string {
	limit = max(1, limit)
	words := strings.Fields(value)
	if len(words) == 0 {
		return nil
	}
	lines := []string{prefix}
	used := 0
	for _, word := range words {
		wordWidth := width(word)
		if used > 0 && used+1+wordWidth > limit {
			lines = append(lines, prefix+word)
			used = wordWidth
			continue
		}
		if used > 0 {
			lines[len(lines)-1] += " "
			used++
		}
		lines[len(lines)-1] += word
		used += wordWidth
	}
	return lines
}

func wrapPreserved(value string, limit int) []string {
	limit = max(1, limit)
	runes := []rune(value)
	if len(runes) == 0 {
		return []string{""}
	}
	lines := make([]string, 0, (len(runes)+limit-1)/limit)
	for len(runes) > 0 {
		count := min(limit, len(runes))
		lines = append(lines, string(runes[:count]))
		runes = runes[count:]
	}
	return lines
}

func width(value string) int {
	visible := 0
	for index := 0; index < len(value); {
		if value[index] == '\x1b' {
			if end := strings.IndexByte(value[index:], 'm'); end >= 0 {
				index += end + 1
				continue
			}
		}
		_, size := utf8.DecodeRuneInString(value[index:])
		visible++
		index += size
	}
	return visible
}
