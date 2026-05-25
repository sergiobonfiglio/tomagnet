package main

import (
	"context"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/sergiobonfiglio/tomagnet/internal/config"
	"github.com/sergiobonfiglio/tomagnet/internal/definitions"
	"github.com/sergiobonfiglio/tomagnet/internal/output"
	"github.com/sergiobonfiglio/tomagnet/internal/search"
	"github.com/sergiobonfiglio/tomagnet/internal/testcmd"
	"github.com/spf13/cobra"
)

var (
	version = "0.3.4"
	commit  = ""
	date    = ""
)

func main() {
	if err := root().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func versionOutput() string {
	parts := []string{"tomagnet " + version}
	if commit != "" {
		parts = append(parts, "commit "+commit)
	}
	if date != "" {
		parts = append(parts, "built "+date)
	}
	return strings.Join(parts, "\n")
}

type modeValue struct{ target *string }

func (m modeValue) String() string {
	if m.target == nil {
		return ""
	}
	return *m.target
}

func (m modeValue) Set(v string) error {
	if slices.Contains(
		[]string{"search", "tv-search", "movie-search", "music-search", "book-search"},
		v,
	) {
		*m.target = v
		return nil
	}
	return fmt.Errorf("invalid mode %q", v)
}

func (m modeValue) Type() string { return "mode" }

func root() *cobra.Command {
	var debug bool
	dbg := func(f string, a ...any) {
		if debug {
			fmt.Fprintf(os.Stderr, f+"\n", a...)
		}
	}

	cmd := &cobra.Command{Use: "tomagnet", Version: version}
	cmd.SetVersionTemplate(versionOutput() + "\n")
	cmd.PersistentFlags().BoolVar(&debug, "debug", false, "debug logs to stderr")

	var indexers, categories []string
	var limit, conc int
	var out, mode, season, episode, imdbid, tmdbid, tvdbid, doubanid, tvmazeid, artist, album, author, title, genre, year string

	searchCmd := &cobra.Command{
		Use:  "search [query]",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := config.Load("")
			if err != nil {
				return err
			}
			idx, err := c.Enabled(indexers)
			if err != nil {
				return err
			}
			if conc == 0 {
				conc = c.Concurrency
			}

			query := ""
			if len(args) > 0 {
				query = args[0]
			}

			r := search.Run(
				context.Background(),
				search.Options{
					Query:       query,
					Mode:        mode,
					Season:      season,
					Episode:     episode,
					IMDBID:      imdbid,
					TMDBID:      tmdbid,
					TVDBID:      tvdbid,
					DoubanID:    doubanid,
					TVMazeID:    tvmazeid,
					Artist:      artist,
					Album:       album,
					Author:      author,
					Title:       title,
					Genre:       genre,
					Year:        year,
					Categories:  categories,
					Indexers:    idx,
					Limit:       limit,
					Concurrency: conc,
					Debug:       dbg,
				},
			)

			if out == "table" {
				return output.Table(os.Stdout, r)
			}
			return output.JSON(os.Stdout, r)
		},
	}

	searchCmd.Flags().StringArrayVar(&indexers, "indexer", nil, "enabled indexer id filter, repeatable")
	searchCmd.Flags().StringArrayVar(&categories, "category", nil, "category filter, repeatable")
	searchCmd.Flags().IntVar(&limit, "limit", 0, "per-indexer result limit")
	searchCmd.Flags().IntVar(&conc, "concurrency", 0, "concurrent indexers")
	searchCmd.Flags().StringVar(&out, "output", "json", "json|table")
	mode = "search"
	searchCmd.Flags().Var(
		modeValue{target: &mode},
		"mode",
		strings.Join(
			[]string{"search", "tv-search", "movie-search", "music-search", "book-search"},
			"|",
		),
	)
	searchCmd.Flags().StringVar(&season, "season", "", "tv-search season")
	searchCmd.Flags().StringVar(&episode, "episode", "", "tv-search episode")
	searchCmd.Flags().StringVar(&imdbid, "imdbid", "", "external imdb id")
	searchCmd.Flags().StringVar(&tmdbid, "tmdbid", "", "external tmdb id")
	searchCmd.Flags().StringVar(&tvdbid, "tvdbid", "", "external tvdb id")
	searchCmd.Flags().StringVar(&doubanid, "doubanid", "", "external douban id")
	searchCmd.Flags().StringVar(&tvmazeid, "tvmazeid", "", "external tvmaze id")
	searchCmd.Flags().StringVar(&artist, "artist", "", "music-search artist")
	searchCmd.Flags().StringVar(&album, "album", "", "music-search album")
	searchCmd.Flags().StringVar(&author, "author", "", "book-search author")
	searchCmd.Flags().StringVar(&title, "title", "", "book-search title")
	searchCmd.Flags().StringVar(&genre, "genre", "", "mode-aware genre")
	searchCmd.Flags().StringVar(&year, "year", "", "mode-aware year")

	defs := &cobra.Command{Use: "definitions"}
	defs.AddCommand(&cobra.Command{
		Use: "sync",
		RunE: func(cmd *cobra.Command, args []string) error {
			m, err := definitions.Sync()
			if err != nil {
				return err
			}
			fmt.Fprintf(os.Stdout, "synced %d definitions to %s\n", len(m.Files), definitions.CacheDir)
			return nil
		},
	})

	test := &cobra.Command{
		Use:  "test [indexer-id]",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := config.Load("")
			if err != nil {
				return err
			}
			ids := args
			idx, err := c.Enabled(ids)
			if err != nil {
				return err
			}
			if !testcmd.Run(context.Background(), os.Stdout, idx, dbg) {
				return fmt.Errorf("no indexer passed")
			}
			return nil
		},
	}

	cmd.AddCommand(searchCmd, defs, test)
	return cmd
}
