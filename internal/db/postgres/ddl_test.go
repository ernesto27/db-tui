package postgres

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildTableDDL(t *testing.T) {
	metadata := tableDDLMetadata{
		schema:    "public",
		tableName: "Album",
		columns: []ddlColumn{
			{Name: "AlbumId", Type: "int4", NotNull: true},
			{Name: "Title", Type: "varchar(160)", NotNull: true},
			{Name: "ArtistId", Type: "int4", NotNull: true},
		},
		constraints: []ddlConstraint{
			{Name: "PK_Album", Definition: `PRIMARY KEY ("AlbumId")`},
			{Name: "FK_AlbumArtistId", Definition: `FOREIGN KEY ("ArtistId") REFERENCES public."Artist"("ArtistId")`},
		},
		indexes: []string{`CREATE INDEX "IFK_AlbumArtistId" ON public."Album" USING btree ("ArtistId")`},
	}

	got, err := buildTableDDL(metadata)

	assert.NoError(t, err)
	assert.Equal(t, `CREATE TABLE "public"."Album" (
    "AlbumId" int4 NOT NULL,
    "Title" varchar(160) NOT NULL,
    "ArtistId" int4 NOT NULL,
    CONSTRAINT "PK_Album" PRIMARY KEY ("AlbumId"),
    CONSTRAINT "FK_AlbumArtistId" FOREIGN KEY ("ArtistId") REFERENCES public."Artist"("ArtistId")
);

CREATE INDEX "IFK_AlbumArtistId" ON public."Album" USING btree ("ArtistId");`, got)
}

func TestBuildTableDDLRejectsUnsupportedRelation(t *testing.T) {
	_, err := buildTableDDL(tableDDLMetadata{tableName: "Partition", unsupported: true})

	assert.EqualError(t, err, "unsupported PostgreSQL table structure")
}

func TestBuildTableDDLIncludesColumnClauses(t *testing.T) {
	got, err := buildTableDDL(tableDDLMetadata{
		schema:    "public",
		tableName: "ddl_review_demo",
		columns: []ddlColumn{
			{Name: "id", Type: "int4", NotNull: true, Default: `nextval('public.ddl_review_demo_id_seq'::regclass)`},
			{Name: "created_at", Type: "timestamp with time zone", NotNull: true, Default: "now()"},
			{Name: "label", Type: "varchar(20)", Collation: `"C"`, Default: `'x'::character varying`},
			{Name: "total", Type: "int4", Generated: "s", Default: "(id * 2)"},
			{Name: "external_id", Type: "int4", Identity: "a"},
		},
	})

	assert.NoError(t, err)
	assert.Contains(t, got, `"id" int4 DEFAULT nextval('public.ddl_review_demo_id_seq'::regclass) NOT NULL`)
	assert.Contains(t, got, `"created_at" timestamp with time zone DEFAULT now() NOT NULL`)
	assert.Contains(t, got, `"label" varchar(20) COLLATE "C" DEFAULT 'x'::character varying`)
	assert.Contains(t, got, `"total" int4 GENERATED ALWAYS AS ((id * 2)) STORED`)
	assert.Contains(t, got, `"external_id" int4 GENERATED ALWAYS AS IDENTITY`)
}
