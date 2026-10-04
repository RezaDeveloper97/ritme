# SEO — structured data

One `<script type="application/ld+json">` per page, rendered by `<x-seo.head/>` from the request's
`App\Domain\Seo\Schema\SchemaGraph`. Pages never print their own JSON-LD (the legacy `seo.jsonld` stack is not for
schema). Validate with Google's Rich Results Test and validator.schema.org after wiring a page.

## Automatic nodes (every page)

| `@id` | Type | Source |
|---|---|---|
| `{site}/#organization` | Organization | settings: name, alternateName, legalName, logo (media), sameAs (social + organization), contactPoint (organization → contact fallback) |
| `{site}/#website` | WebSite | settings; `SearchAction` once `searchUrlTemplate()` is set (L4-04) |
| `{canonical}#webpage` | WebPage / AboutPage / ContactPage / CollectionPage / FAQPage / ItemPage … | SeoHead title, description, OG image; `pageType()`, `dates()`, `reviewedBy()` |
| `{canonical}#breadcrumb` | BreadcrumbList | `breadcrumbs()` (same items as `x-ui.breadcrumbs`), else Home → page; none on the home page |

Pages add nodes and page facts **before the head renders** (controller, or the page view's sections / slot).
Admin `seo_meta.schema_overrides`: `{"#webpage": {…merged props…}, "@graph": [{…extra node…}]}`.

## Page builders (`Schema/Nodes`) and Google requirements

| Builder | Google required | Recommended (we emit when known) | Notes |
|---|---|---|---|
| `BreadcrumbListNode` | itemListElement ≥ 1 ListItem: position, name, item | ≥ 2 items | last item may omit URL (we use the page URL) |
| `OrganizationNode` | — (no required props) | name, url, logo (≥ 112×112), sameAs, contactPoint | logo from `organization.logo_media_id` once L2 resolves media |
| `MobileApplicationNode` | name, offers.price (0), aggregateRating **or** review | applicationCategory, operatingSystem | no rating → valid but no rich result; **never invent ratings** |
| `FaqPageNode` | mainEntity Question.name + acceptedAnswer.text | — | only visible, site-authored Q&A; Google limits FAQ results to authoritative health/gov sites |
| `BlogPostingNode` | — | headline (≤ 110), image (≥ 1200 px wide), datePublished, dateModified, author.name + url | YMYL: `reviewedBy()` + `lastReviewed` on the page node, author/reviewer with credential |
| `ProductNode` | name + one of offers / review / aggregateRating | image, sku, brand, offers.price/priceCurrency (IRR)/availability | rating only from real on-site reviews (`AggregateRatingData` rejects count 0) |
| `LocalBusinessNode` | name, address | geo (≥ 5 decimals), telephone, openingHoursSpecification, image, url, priceRange | most specific subtype (`ChildCare`, `SportsActivityLocation`, `MedicalClinic` …) |
| `ItemListNode` | itemListElement: ListItem position + url | name | summary-page form for listing pages |

Prices are in rial (`IRR`): convert toman × 10. Dates are ISO 8601 with offset (`+03:30`).
