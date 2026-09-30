package musicaldna

// ProjectDNA is Sound Continuum's stable, project-wide editorial identity —
// "what is Sound Continuum?" — as opposed to WeeklyDirection, which answers
// "what is this particular edition becoming?" and changes every week.
//
// docs/manifesto.md and Sound Continuum's M1 decisions are ProjectDNA's only
// source. They describe editorial process and philosophy (music as a
// continuum, musical bridges over individual merit, discovery without
// forced obscurity, human editorial judgment) rather than concrete
// mood/energy/texture/cultural values — so Profile stays entirely unset by
// default (see DefaultProjectDNA). Setting a Profile field here would assert
// "Sound Continuum's identity always has this musical quality," which is not
// what the manifesto claims and would contradict its continuum principle.
// A field should only ever be set here once a durable, project-wide trait is
// actually documented in docs/manifesto.md or docs/memory/decisions.md — see
// the Card #41 entry there for the current (empty) mapping.
type ProjectDNA struct {
	Profile Profile
}

// DefaultProjectDNA returns Sound Continuum's current ProjectDNA: every
// Profile dimension unset. This is not a placeholder awaiting future
// implementation of this package — it is the accurate reflection of what
// docs/manifesto.md actually specifies today.
func DefaultProjectDNA() ProjectDNA {
	return ProjectDNA{}
}
