package healthrecord

// Provider sections: report sections built by a SectionProvider of another package (wired with WithProviders), so the
// record package never imports them. The doctor report request accepts their keys next to the built-in ones; a key
// whose provider is not wired, or whose provider leaves the section out for this user, is simply absent from the
// record.
const (
	// SectionMenopause is the menopause report section (roadmap CB-MENO-03, internal/menopause.ReportSection):
	// stage, score first → last, hot flashes, night sweats, sleep, bleeding, blood pressure, top symptoms, treatment
	// adherence, side effects and supplements over the report window. Menopause mode only.
	SectionMenopause = "menopause"
)

// ProviderSections are the provider section keys, in report order (after the built-in sections).
var ProviderSections = []string{SectionMenopause}

// ReportSections are the section keys a report request may select: the built-in sections, then the provider ones.
func ReportSections() []string {
	out := make([]string, 0, len(Sections)+len(ProviderSections))
	out = append(out, Sections...)
	return append(out, ProviderSections...)
}
