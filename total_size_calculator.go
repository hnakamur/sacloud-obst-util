package main

import (
	"io"

	"github.com/orisano/gosax"
)

type sizeOnlyContents struct {
	size uint64
}

type onSizeOnlyContentsEndFunc func(contents sizeOnlyContents) error

type totalSizeCalculator struct {
	handleEvent   handleEventFunc
	handleText    handleTextFunc
	onContentsEnd onSizeOnlyContentsEndFunc

	nextContinuationToken string
	keyCount              uint64
	isTruncated           bool

	currentContents sizeOnlyContents

	objCount  uint64
	totalSize uint64
}

func newTotalSizeCalculator() *totalSizeCalculator {
	p := &totalSizeCalculator{}
	p.handleEvent = p.handleEventStart
	p.onContentsEnd = p.handleBuiltContents
	return p
}

func (h *totalSizeCalculator) handleResponseBody(reader io.Reader) error {
	r := gosax.NewReader(reader)
	for {
		e, err := r.Event()
		if err != nil {
			return err
		}
		if e.Type() == gosax.EventEOF {
			break
		}
		if err := h.handleEvent(e); err != nil {
			return err
		}
	}
	return nil
}

func (h *totalSizeCalculator) handleEventStart(e gosax.Event) error {
	if e.Type() == gosax.EventStart {
		elem, err := gosax.StartElement(e.Bytes)
		if err != nil {
			return err
		}
		switch elem.Name.Local {
		case elementNameNextContinuationToken:
			if isSelfClosing(e) {
				h.nextContinuationToken = ""
			} else {
				h.handleEvent = h.buildHandleEventText(h.handleEventEnd)
				h.handleText = buildHandleTextString(&h.nextContinuationToken)
			}
		case elementNameKeyCount:
			h.handleEvent = h.buildHandleEventText(h.handleEventEnd)
			h.handleText = buildHandleTextUint64(&h.keyCount)
		case elementNameIsTruncated:
			h.handleEvent = h.buildHandleEventText(h.handleEventEnd)
			h.handleText = buildHandleTextBool(&h.isTruncated)
		case elementNameContents:
			h.handleEvent = h.handleContentsEventStartOrEnd
		}
	}
	return nil
}

func (h *totalSizeCalculator) buildHandleEventText(nextHandleEvent handleEventFunc) handleEventFunc {
	return func(e gosax.Event) error {
		if e.Type() == gosax.EventText {
			if h.handleText != nil {
				if err := h.handleText(e.Bytes); err != nil {
					return err
				}
				h.handleText = nil
			}
			h.handleEvent = nextHandleEvent
		}
		return nil
	}
}

func (h *totalSizeCalculator) handleEventEnd(e gosax.Event) error {
	if e.Type() == gosax.EventEnd {
		h.handleEvent = h.handleEventStart
	}
	return nil
}

func (h *totalSizeCalculator) handleContentsEventStartOrEnd(e gosax.Event) error {
	switch e.Type() {
	case gosax.EventStart:
		elem, err := gosax.StartElement(e.Bytes)
		if err != nil {
			return err
		}
		switch elem.Name.Local {
		case elementNameSize:
			h.handleEvent = h.buildHandleEventText(h.handleContentsEventStartOrEnd)
			h.handleText = buildHandleTextUint64(&h.currentContents.size)
		}
	case gosax.EventEnd:
		elem := gosax.EndElement(e.Bytes)
		if elem.Name.Local == elementNameContents {
			if err := h.onContentsEnd(h.currentContents); err != nil {
				return err
			}
			h.handleEvent = h.handleEventStart
		}
	}
	return nil
}

func (h *totalSizeCalculator) handleBuiltContents(contents sizeOnlyContents) error {
	h.objCount++
	h.totalSize += contents.size
	return nil
}
