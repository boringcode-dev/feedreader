package sources

import (
	"strings"
	"testing"
	"time"

	"github.com/PuerkitoBio/goquery"
)

func TestParseAlphaXivExploreExtractsPaperFields(t *testing.T) {
	payload := `
	<div class="rounded-xl border-[0.5px] border-border bg-bg px-4 py-3 backdrop-blur-sm transition-all hover:shadow-md">
	  <div class="flex w-full gap-6">
	    <div class="flex min-w-0 flex-1 flex-col gap-4">
	      <a data-loading-trigger="true" href="/abs/2606.15956" target="_self">
	        <div class="box-border grid w-full overflow-hidden text-left wrap-break-word whitespace-normal">
	          <div class="tiptap html-renderer box-border w-full overflow-hidden text-left wrap-break-word whitespace-normal text-[22px] leading-tight font-bold text-text transition-all hover:underline">You Don't Need Strong Assumptions: Visual Representation Learning via Temporal Differences</div>
	        </div>
	      </a>
	      <div class="flex items-center gap-4">
	        <span class="text-sm font-medium whitespace-nowrap text-text">14 Jun 2026</span>
	        <div class="relative min-w-0 overflow-hidden">
	          <div class="scrollbar-hide flex items-center gap-4 overflow-x-auto mask-fade-x mask-fade-x-start-0 mask-fade-x-end-0">
	            <div class="flex shrink-0 items-center gap-3 text-sm">
	              <div class="flex items-center gap-1.5 font-normal">Ninad Daithankar</div>
	              <div class="flex items-center gap-1.5 font-normal">Alexi Gladstone</div>
	              <div class="flex items-center gap-1.5 font-normal">Yann LeCun</div>
	            </div>
	          </div>
	        </div>
	      </div>
	      <div class="flex flex-col gap-1">
	        <p class="line-clamp-4 text-xs/normal font-normal tracking-wide text-subtext">
	          <svg class="lucide lucide-sparkles"></svg>
	          <span>Researchers from UIUC and NYU propose Temporal Difference in Vision (TDV), a self-supervised method for learning visual representations from video by predicting future frame embeddings from past frames and learned motion.</span>
	        </p>
	        <a href="/overview/2606.15956" class="inline-flex items-center text-xs font-semibold text-blue hover:underline">View blog</a>
	      </div>
	      <div class="flex items-center justify-between">
	        <div class="scrollbar-hide flex items-center gap-4 overflow-x-auto mask-fade-x mask-fade-x-start-0 mask-fade-x-end-0">
	          <a data-loading-trigger="true" href="/?subcategories=%5B%22artificial-intelligence%22%5D" class="shrink-0 cursor-pointer text-xs font-medium text-text transition-colors hover:text-custom-red">#artificial-intelligence</a>
	          <a data-loading-trigger="true" href="/?subcategories=%5B%22computer-vision-and-pattern-recognition%22%5D" class="shrink-0 cursor-pointer text-xs font-medium text-text transition-colors hover:text-custom-red">#computer-vision-and-pattern-recognition</a>
	          <a data-loading-trigger="true" href="/?subcategories=%5B%22machine-learning%22%5D" class="shrink-0 cursor-pointer text-xs font-medium text-text transition-colors hover:text-custom-red">#machine-learning</a>
	        </div>
	      </div>
	      <div class="mt-auto flex items-center justify-between gap-2">
	        <div class="scrollbar-hide flex min-w-0 flex-1 items-center gap-4 overflow-x-auto mask-fade-x mask-fade-x-start-0 mask-fade-x-end-0">
	          <button class="cursor-pointer items-center gap-1.5 text-sm transition-colors flex h-8 shrink-0 rounded-full px-2.5 py-1.5 font-normal bg-surface text-text">
	            <div class="interactable-overlay bg-overlay"></div>
	            <svg class="lucide lucide-thumbs-up" aria-hidden="true"></svg>
	            <span class="inline-block">86</span>
	          </button>
	          <a data-loading-trigger="true" href="/replicate/2606.15956" target="_self" class="flex h-8 shrink-0 items-center gap-1.5 rounded-full bg-surface px-2.5 py-1.5 text-sm font-normal text-text transition-colors"><span>Run now</span></a>
	          <a data-loading-trigger="true" href="/audio/2606.15956" target="_self" class="flex h-8 shrink-0 items-center gap-1.5 rounded-full bg-surface px-2.5 py-1.5 text-sm font-normal text-text transition-colors"><span>Audio</span></a>
	        </div>
	      </div>
	    </div>
	  </div>
	</div>`

	items, err := parseAlphaXivExplore(strings.NewReader(payload))
	if err != nil {
		t.Fatalf("parse alphaXiv payload: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}

	item := items[0]
	if item.Source != "alphaxiv" {
		t.Fatalf("unexpected source: %q", item.Source)
	}
	if item.ExternalID != "2606.15956" {
		t.Fatalf("unexpected external id: %q", item.ExternalID)
	}
	if item.URL != "https://www.alphaxiv.org/abs/2606.15956" {
		t.Fatalf("unexpected url: %q", item.URL)
	}
	if item.Title != "You Don't Need Strong Assumptions: Visual Representation Learning via Temporal Differences" {
		t.Fatalf("unexpected title: %q", item.Title)
	}
	if item.Summary == nil || !strings.Contains(*item.Summary, "Temporal Difference in Vision") {
		t.Fatalf("unexpected summary: %#v", item.Summary)
	}
	if item.Author == nil || *item.Author != "Ninad Daithankar, Alexi Gladstone, Yann LeCun" {
		t.Fatalf("unexpected author: %#v", item.Author)
	}
	if item.Score == nil || *item.Score != 86 {
		t.Fatalf("unexpected score: %#v", item.Score)
	}
	if item.PublishedAt == nil {
		t.Fatal("expected publishedAt")
	}
	wantPublishedAt := time.Date(2026, time.June, 14, 0, 0, 0, 0, time.UTC)
	if !item.PublishedAt.Equal(wantPublishedAt) {
		t.Fatalf("unexpected publishedAt: got %s want %s", item.PublishedAt.Format(time.RFC3339), wantPublishedAt.Format(time.RFC3339))
	}
	if item.SourceRank != 1 {
		t.Fatalf("unexpected source rank: %d", item.SourceRank)
	}

	tags, ok := item.Metadata["tags"].([]string)
	if !ok {
		t.Fatalf("expected []string tags metadata, got %#v", item.Metadata["tags"])
	}
	if len(tags) != 3 || tags[0] != "artificial-intelligence" || tags[2] != "machine-learning" {
		t.Fatalf("unexpected tags: %#v", tags)
	}
}

func TestParseAlphaXivExploreSkipsCardsWithoutAbsLink(t *testing.T) {
	payload := `
	<div>
	  <a href="/overview/2606.15956">View blog</a>
	  <div>Not a paper card</div>
	</div>`

	items, err := parseAlphaXivExplore(strings.NewReader(payload))
	if err != nil {
		t.Fatalf("parse alphaXiv payload: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected 0 items, got %d", len(items))
	}
}

func TestAlphaXivPublishedAtNormalizesWhitespace(t *testing.T) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(`<div>14&nbsp;Jun&nbsp;2026</div>`))
	if err != nil {
		t.Fatalf("build document: %v", err)
	}
	publishedAt := alphaXivPublishedAt(doc.Selection)
	if publishedAt == nil {
		t.Fatal("expected publishedAt")
	}
	want := time.Date(2026, time.June, 14, 0, 0, 0, 0, time.UTC)
	if !publishedAt.Equal(want) {
		t.Fatalf("unexpected publishedAt: got %s want %s", publishedAt.Format(time.RFC3339), want.Format(time.RFC3339))
	}
}
