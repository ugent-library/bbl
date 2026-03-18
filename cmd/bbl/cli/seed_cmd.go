package cli

import (
	"fmt"
	"iter"
	"log/slog"
	"time"

	"github.com/spf13/cobra"
	"github.com/ugent-library/bbl"
)

const seedSource = "seed"

// seedRecord is a placeholder source record for seed data.
var seedRecord = []byte("{}")

func newSeedCmd(e *env) *cobra.Command {
	return &cobra.Command{
		Use:   "seed",
		Short: "Populate the database with development data",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			svc, err := e.services(ctx)
			if err != nil {
				return err
			}
			defer svc.Repo.Close()

			repo := svc.Repo

			// Register seed source.
			if err := repo.UpsertSource(ctx, seedSource); err != nil {
				return err
			}

			// --- Organizations ---
			slog.Info("seeding organizations")
			orgs := seedOrganizations()
			for _, o := range orgs {
				o.SourceRecord = seedRecord
			}
			n, err := repo.ImportOrganizations(ctx, seedSource, seqOf(orgs))
			if err != nil {
				return fmt.Errorf("seed organizations: %w", err)
			}
			slog.Info("seeded organizations", "count", n)

			// --- People ---
			slog.Info("seeding people")
			people := seedPeople()
			for _, p := range people {
				p.SourceRecord = seedRecord
			}
			n, err = repo.ImportPeople(ctx, seedSource, seqOf(people))
			if err != nil {
				return fmt.Errorf("seed people: %w", err)
			}
			slog.Info("seeded people", "count", n)

			// --- Projects ---
			slog.Info("seeding projects")
			projects := seedProjects(people)
			for _, p := range projects {
				p.SourceRecord = seedRecord
			}
			n, err = repo.ImportProjects(ctx, seedSource, seqOf(projects))
			if err != nil {
				return fmt.Errorf("seed projects: %w", err)
			}
			slog.Info("seeded projects", "count", n)

			// --- Works ---
			slog.Info("seeding works")
			works := seedWorks(people, projects)
			for _, w := range works {
				w.SourceRecord = seedRecord
			}
			n, err = repo.ImportWorks(ctx, seedSource, seqOf(works))
			if err != nil {
				return fmt.Errorf("seed works: %w", err)
			}
			slog.Info("seeded works", "count", n)

			// --- Users ---
			slog.Info("seeding users")
			for _, u := range seedUsers() {
				if _, err := repo.CreateUser(ctx, u); err != nil {
					return fmt.Errorf("seed users: %w", err)
				}
			}
			slog.Info("seeded users")

			// --- Reindex ---
			if svc.Index != nil {
				slog.Info("reindexing")
				for _, reindex := range []struct {
					name string
					fn   func() error
				}{
					{"organizations", func() error {
						return svc.Index.Organizations().Reindex(ctx, repo.EachOrganization(ctx), func(since time.Time) iter.Seq2[*bbl.Organization, error] {
							return repo.EachOrganizationSince(ctx, since)
						})
					}},
					{"people", func() error {
						return svc.Index.People().Reindex(ctx, repo.EachPerson(ctx), func(since time.Time) iter.Seq2[*bbl.Person, error] {
							return repo.EachPersonSince(ctx, since)
						})
					}},
					{"projects", func() error {
						return svc.Index.Projects().Reindex(ctx, repo.EachProject(ctx), func(since time.Time) iter.Seq2[*bbl.Project, error] {
							return repo.EachProjectSince(ctx, since)
						})
					}},
					{"works", func() error {
						return svc.Index.Works().Reindex(ctx, repo.EachWork(ctx), func(since time.Time) iter.Seq2[*bbl.Work, error] {
							return repo.EachWorkSince(ctx, since)
						})
					}},
				} {
					slog.Info("reindexing", "entity", reindex.name)
					if err := reindex.fn(); err != nil {
						return fmt.Errorf("reindex %s: %w", reindex.name, err)
					}
				}
				slog.Info("reindex complete")
			}

			slog.Info("seed complete")
			return nil
		},
	}
}

// seqOf converts a slice into an iter.Seq2 suitable for Import* methods.
func seqOf[T any](items []*T) iter.Seq2[*T, error] {
	return func(yield func(*T, error) bool) {
		for _, item := range items {
			if !yield(item, nil) {
				return
			}
		}
	}
}

// --- Seed data ---

func seedOrganizations() []*bbl.ImportOrganizationInput {
	return []*bbl.ImportOrganizationInput{
		{
			SourceID: "ugent",
			Kind:     "university",
			Names: []bbl.Text{
				{Lang: "eng", Val: "Ghent University"},
				{Lang: "dut", Val: "Universiteit Gent"},
			},
			Identifiers: []bbl.Identifier{
				{Scheme: "ror", Val: "https://ror.org/00cv9y106"},
			},
		},
		{
			SourceID: "dept-cs",
			Kind:     "department",
			Names: []bbl.Text{
				{Lang: "eng", Val: "Department of Computer Science"},
				{Lang: "dut", Val: "Vakgroep Informatica"},
			},
			Rels: []bbl.ImportOrganizationRel{
				{
					Ref:  bbl.Ref{SourceID: "ugent"},
					Kind: "part_of",
				},
			},
		},
		{
			SourceID: "dept-history",
			Kind:     "department",
			Names: []bbl.Text{
				{Lang: "eng", Val: "Department of History"},
				{Lang: "dut", Val: "Vakgroep Geschiedenis"},
			},
			Rels: []bbl.ImportOrganizationRel{
				{
					Ref:  bbl.Ref{SourceID: "ugent"},
					Kind: "part_of",
				},
			},
		},
	}
}

func seedPeople() []*bbl.ImportPersonInput {
	return []*bbl.ImportPersonInput{
		{
			SourceID:   "p-einstein",
			Name:       "Albert Einstein",
			GivenName:  "Albert",
			FamilyName: "Einstein",
			Identifiers: []bbl.Identifier{
				{Scheme: "orcid", Val: "0000-0001-0001-0001"},
			},
			Affiliations: []bbl.ImportPersonAffiliation{
				{Ref: bbl.Ref{SourceID: "dept-cs"}},
			},
		},
		{
			SourceID:   "p-curie",
			Name:       "Marie Curie",
			GivenName:  "Marie",
			FamilyName: "Curie",
			Identifiers: []bbl.Identifier{
				{Scheme: "orcid", Val: "0000-0001-0001-0002"},
			},
			Affiliations: []bbl.ImportPersonAffiliation{
				{Ref: bbl.Ref{SourceID: "dept-cs"}},
			},
		},
		{
			SourceID:   "p-turing",
			Name:       "Alan Turing",
			GivenName:  "Alan",
			MiddleName: "Mathison",
			FamilyName: "Turing",
			Identifiers: []bbl.Identifier{
				{Scheme: "orcid", Val: "0000-0001-0001-0003"},
			},
			Affiliations: []bbl.ImportPersonAffiliation{
				{Ref: bbl.Ref{SourceID: "dept-cs"}},
			},
		},
		{
			SourceID:   "p-noether",
			Name:       "Emmy Noether",
			GivenName:  "Emmy",
			FamilyName: "Noether",
			Identifiers: []bbl.Identifier{
				{Scheme: "orcid", Val: "0000-0001-0001-0004"},
			},
		},
		{
			SourceID:   "p-braudel",
			Name:       "Fernand Braudel",
			GivenName:  "Fernand",
			FamilyName: "Braudel",
			Affiliations: []bbl.ImportPersonAffiliation{
				{Ref: bbl.Ref{SourceID: "dept-history"}},
			},
		},
	}
}

func seedProjects(people []*bbl.ImportPersonInput) []*bbl.ImportProjectInput {
	return []*bbl.ImportProjectInput{
		{
			SourceID:     "proj-quantum",
			Status:       "public",
			Titles:       []bbl.Title{{Lang: "eng", Val: "Quantum Foundations and Applications"}},
			Descriptions: []bbl.Text{{Lang: "eng", Val: "Investigating foundational questions in quantum mechanics and their applications to computing."}},
			Participants: []bbl.ImportProjectParticipant{
				{Ref: bbl.Ref{SourceID: "p-einstein"}, Role: "principal_investigator"},
				{Ref: bbl.Ref{SourceID: "p-noether"}, Role: "co_investigator"},
			},
		},
		{
			SourceID:     "proj-computation",
			Status:       "public",
			Titles:       []bbl.Title{{Lang: "eng", Val: "Foundations of Computation"}},
			Descriptions: []bbl.Text{{Lang: "eng", Val: "Exploring the theoretical limits of computation and decidability."}},
			Participants: []bbl.ImportProjectParticipant{
				{Ref: bbl.Ref{SourceID: "p-turing"}, Role: "principal_investigator"},
			},
		},
		{
			SourceID:     "proj-mediterranean",
			Status:       "public",
			Titles:       []bbl.Title{{Lang: "eng", Val: "The Mediterranean World in the Early Modern Period"}},
			Descriptions: []bbl.Text{{Lang: "eng", Val: "A longue durée study of Mediterranean trade networks and cultural exchange."}},
			Participants: []bbl.ImportProjectParticipant{
				{Ref: bbl.Ref{SourceID: "p-braudel"}, Role: "principal_investigator"},
			},
		},
	}
}

func seedWorks(_ []*bbl.ImportPersonInput, _ []*bbl.ImportProjectInput) []*bbl.ImportWorkInput {
	// Lookup tables for generating varied works.
	personIDs := []string{"p-einstein", "p-curie", "p-turing", "p-noether", "p-braudel"}
	projectIDs := []string{"proj-quantum", "proj-computation", "proj-mediterranean"}

	kinds := []string{
		"journal_article", "journal_article", "journal_article", "journal_article", // weighted
		"book", "book_chapter", "conference_paper", "dissertation",
		"edited_book", "preprint",
	}
	statuses := []string{
		"public", "public", "public", "public", "public", // weighted
		"private", "private",
	}
	journals := []string{
		"Annalen der Physik", "Physical Review", "Nature", "Science",
		"Proceedings of the London Mathematical Society", "Mind",
		"Journal of Computational Physics", "Reviews of Modern Physics",
		"Mathematische Annalen", "Comptes Rendus",
		"Physical Review Letters", "Journal of Mathematical Physics",
		"Transactions of the American Mathematical Society",
		"Bulletin de la Société Mathématique de France",
		"Philosophical Transactions of the Royal Society",
	}
	publishers := []string{
		"Academic Press", "Cambridge University Press", "Springer",
		"Oxford University Press", "Princeton University Press",
		"Armand Colin", "Wiley", "Elsevier",
	}
	classifications := []string{
		"PHYSICS, MULTIDISCIPLINARY", "PHYSICS, MATHEMATICAL",
		"PHYSICS, NUCLEAR", "MATHEMATICS", "HISTORY",
		"COMPUTER SCIENCE, THEORY & METHODS",
		"COMPUTER SCIENCE, ARTIFICIAL INTELLIGENCE",
		"CHEMISTRY, PHYSICAL", "PHILOSOPHY",
		"ENGINEERING, ELECTRICAL & ELECTRONIC",
	}
	kw := func(vals ...string) []bbl.Keyword {
		kws := make([]bbl.Keyword, len(vals))
		for i, v := range vals {
			kws[i] = bbl.Keyword{Val: v}
		}
		return kws
	}
	topics := []struct {
		title    string
		keywords []bbl.Keyword
	}{
		{"Quantum Field Theory and Renormalization", kw("quantum field theory", "renormalization", "gauge theory")},
		{"Algebraic Structures in Topology", kw("algebra", "topology", "homology")},
		{"Radiation and Matter Interactions", kw("radiation", "matter", "spectroscopy")},
		{"Decidability and Recursive Functions", kw("decidability", "recursive functions", "logic")},
		{"Abstract Algebra and Ring Theory", kw("abstract algebra", "ring theory", "ideals")},
		{"Trade Routes in the Ancient World", kw("trade", "ancient world", "commerce")},
		{"Brownian Motion and Molecular Theory", kw("Brownian motion", "molecules", "diffusion")},
		{"Machine Learning Foundations", kw("machine learning", "statistical learning", "classification")},
		{"Nuclear Fission and Chain Reactions", kw("nuclear fission", "chain reaction", "uranium")},
		{"Cryptanalysis and Code Breaking", kw("cryptanalysis", "cryptography", "ciphers")},
		{"Representation Theory of Groups", kw("representation theory", "group theory", "linear algebra")},
		{"Colonial Economies and Trade", kw("colonialism", "economy", "trade networks")},
		{"General Relativity and Cosmology", kw("general relativity", "cosmology", "spacetime")},
		{"Neural Networks and Computation", kw("neural networks", "computation", "pattern recognition")},
		{"Isotope Separation Techniques", kw("isotopes", "separation", "mass spectrometry")},
		{"Formal Verification of Programs", kw("formal verification", "program correctness", "logic")},
		{"Commutative Algebra Foundations", kw("commutative algebra", "modules", "Noetherian rings")},
		{"Maritime History and Navigation", kw("maritime history", "navigation", "seafaring")},
		{"Photoelectric Effect in Metals", kw("photoelectric effect", "metals", "electron emission")},
		{"Automata Theory and Languages", kw("automata", "formal languages", "grammars")},
		{"Invariant Theory and Symmetry", kw("invariant theory", "symmetry", "transformations")},
		{"Urbanization in Early Modern Europe", kw("urbanization", "early modern", "Europe")},
		{"Gravitational Waves Detection", kw("gravitational waves", "LIGO", "interferometry")},
		{"Complexity Classes and Reductions", kw("complexity theory", "NP-completeness", "reductions")},
		{"Homological Algebra Methods", kw("homological algebra", "derived functors", "exact sequences")},
		{"Mediterranean Agriculture", kw("agriculture", "Mediterranean", "climate")},
		{"Unified Field Theory Attempts", kw("unified field theory", "electromagnetism", "gravity")},
		{"Lambda Calculus and Type Theory", kw("lambda calculus", "type theory", "functional programming")},
		{"Radioactive Decay Rates", kw("radioactive decay", "half-life", "nuclear physics")},
		{"Information Theory Basics", kw("information theory", "entropy", "communication")},
		{"Galois Theory Applications", kw("Galois theory", "field extensions", "solvability")},
		{"Venice and Ottoman Relations", kw("Venice", "Ottoman Empire", "diplomacy")},
		{"Bose-Einstein Condensation", kw("Bose-Einstein", "condensation", "quantum statistics")},
		{"Halting Problem Variations", kw("halting problem", "undecidability", "Turing degrees")},
		{"Tensor Analysis and Manifolds", kw("tensor analysis", "manifolds", "differential geometry")},
		{"Grain Trade in the Mediterranean", kw("grain trade", "Mediterranean", "food supply")},
		{"Quantum Entanglement Experiments", kw("quantum entanglement", "Bell inequality", "experiments")},
		{"Recursive Function Theory", kw("recursive functions", "computability", "primitive recursion")},
		{"Algebraic Number Theory", kw("algebraic number theory", "number fields", "class groups")},
		{"Piracy and Maritime Law", kw("piracy", "maritime law", "corsairs")},
	}
	conferences := []string{
		"International Conference on Quantum Information",
		"Symposium on Theoretical Computer Science",
		"European Mathematical Congress",
		"International Congress of Historians",
		"Conference on Nuclear Physics",
	}
	locations := []string{
		"Geneva, Switzerland", "Berlin, Germany", "Paris, France",
		"Cambridge, UK", "Princeton, USA", "Vienna, Austria",
	}

	var works []*bbl.ImportWorkInput

	for i := 0; i < 150; i++ {
		kind := kinds[i%len(kinds)]
		status := statuses[i%len(statuses)]
		topic := topics[i%len(topics)]
		year := fmt.Sprintf("%d", 1900+i%126) // 1900-2025
		person := personIDs[i%len(personIDs)]
		cls := classifications[i%len(classifications)]

		w := &bbl.ImportWorkInput{
			SourceID: fmt.Sprintf("w-gen-%03d", i),
			Kind:     kind,
			Status:   status,
			Titles:   []bbl.Title{{Lang: "eng", Val: topic.title}},
			Keywords:          topic.keywords,
			PublicationYear:   year,
			PublicationStatus: "published",
			Classifications: []bbl.Identifier{
				{Scheme: "wos", Val: cls},
			},
			Contributors: []bbl.ImportWorkContributor{
				{PersonRef: &bbl.Ref{SourceID: person}, Roles: []string{"author"}},
			},
		}

		// Add a DOI to most works.
		if i%3 != 0 {
			w.Identifiers = []bbl.Identifier{
				{Scheme: "doi", Val: fmt.Sprintf("10.1234/seed.%03d", i)},
			}
		}

		// Add journal details for articles.
		switch kind {
		case "journal_article", "preprint":
			w.JournalTitle = journals[i%len(journals)]
			w.Volume = fmt.Sprintf("%d", 1+i%50)
			w.Issue = fmt.Sprintf("%d", 1+i%12)
			startPage := 100 + i*7
			w.Pages = bbl.Extent{
				Start: fmt.Sprintf("%d", startPage),
				End:   fmt.Sprintf("%d", startPage+10+i%20),
			}
		case "book", "edited_book":
			w.Publisher = publishers[i%len(publishers)]
			w.PlaceOfPublication = locations[i%len(locations)]
			w.Identifiers = []bbl.Identifier{
				{Scheme: "isbn", Val: fmt.Sprintf("978-0-00-%06d-%d", i, i%10)},
			}
		case "book_chapter":
			w.Publisher = publishers[i%len(publishers)]
			w.BookTitle = fmt.Sprintf("Collected Studies in %s", topic.keywords[0].Val)
			startPage := 10 + i%200
			w.Pages = bbl.Extent{
				Start: fmt.Sprintf("%d", startPage),
				End:   fmt.Sprintf("%d", startPage+15),
			}
		case "conference_paper":
			w.Conference = bbl.Conference{
				Name:     conferences[i%len(conferences)],
				Location: locations[i%len(locations)],
			}
		case "dissertation":
			w.Publisher = "University Press"
			w.PlaceOfPublication = locations[i%len(locations)]
		}

		// Add abstracts to ~60% of works.
		if i%5 != 0 {
			w.Abstracts = []bbl.Text{
				{Lang: "eng", Val: fmt.Sprintf("This paper investigates %s in the context of %s.",
					topic.keywords[0].Val, topic.keywords[len(topic.keywords)-1].Val)},
			}
		}

		// Add a second contributor to ~30% of works.
		if i%3 == 0 {
			second := personIDs[(i+1)%len(personIDs)]
			w.Contributors = append(w.Contributors,
				bbl.ImportWorkContributor{PersonRef: &bbl.Ref{SourceID: second}, Roles: []string{"author"}},
			)
		}

		// Add a second classification to ~25% of works.
		if i%4 == 0 {
			w.Classifications = append(w.Classifications,
				bbl.Identifier{Scheme: "wos", Val: classifications[(i+3)%len(classifications)]},
			)
		}

		// Link ~40% of works to a project.
		if i%5 < 2 {
			w.Projects = []bbl.ImportWorkProject{
				{Ref: bbl.Ref{SourceID: projectIDs[i%len(projectIDs)]}},
			}
		}

		// Editors instead of authors for edited books.
		if kind == "edited_book" {
			for j := range w.Contributors {
				w.Contributors[j].Roles = []string{"editor"}
			}
		}

		works = append(works, w)
	}

	return works
}

func seedUsers() []bbl.UserAttrs {
	return []bbl.UserAttrs{
		{Username: "admin", Email: "admin@example.com", Name: "Admin User", Role: "admin"},
		{Username: "researcher", Email: "researcher@example.com", Name: "Jane Researcher", Role: "user"},
	}
}
