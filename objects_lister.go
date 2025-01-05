package main

import (
	"io"

	"github.com/orisano/gosax"
)

type onContentsEndFunc func(contents listBucketsResultContents) (discardsRest bool, err error)

type objectsLister struct {
	handleEvent   handleEventFunc
	handleText    handleTextFunc
	onContentsEnd onContentsEndFunc

	nextContinuationToken string
	keyCount              uint64
	isTruncated           bool

	currentContents listBucketsResultContents

	exitsLoop bool
}

type listBucketsResultContents struct {
	key          string
	lastModified string
	size         uint64
}

func newObjectsLister(onContentsEnd onContentsEndFunc) *objectsLister {
	p := &objectsLister{}
	p.handleEvent = p.handleEventStart
	p.onContentsEnd = onContentsEnd
	return p
}

func (h *objectsLister) handleResponseBody(reader io.Reader) error {
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
		if h.exitsLoop {
			break
		}
	}
	return nil
}

func (h *objectsLister) handleEventStart(e gosax.Event) error {
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

func (h *objectsLister) buildHandleEventText(nextHandleEvent handleEventFunc) handleEventFunc {
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

func (h *objectsLister) handleEventEnd(e gosax.Event) error {
	if e.Type() == gosax.EventEnd {
		h.handleEvent = h.handleEventStart
	}
	return nil
}

func (h *objectsLister) handleContentsEventStartOrEnd(e gosax.Event) error {
	switch e.Type() {
	case gosax.EventStart:
		elem, err := gosax.StartElement(e.Bytes)
		if err != nil {
			return err
		}
		switch elem.Name.Local {
		case elementNameKey:
			h.handleEvent = h.buildHandleEventText(h.handleContentsEventStartOrEnd)
			h.handleText = buildHandleTextString(&h.currentContents.key)
		case elementNameLastModified:
			h.handleEvent = h.buildHandleEventText(h.handleContentsEventStartOrEnd)
			h.handleText = buildHandleTextString(&h.currentContents.lastModified)
		case elementNameSize:
			h.handleEvent = h.buildHandleEventText(h.handleContentsEventStartOrEnd)
			h.handleText = buildHandleTextUint64(&h.currentContents.size)
		}
	case gosax.EventEnd:
		elem := gosax.EndElement(e.Bytes)
		if elem.Name.Local == elementNameContents {
			discardsRest, err := h.onContentsEnd(h.currentContents)
			if err != nil {
				return err
			}
			h.exitsLoop = discardsRest
			h.handleEvent = h.handleEventStart
		}
	}
	return nil
}
