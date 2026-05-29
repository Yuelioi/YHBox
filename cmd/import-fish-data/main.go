// cmd/import-fish-data/main.go
//
// import-fish-data 一次性迁移工具:
//   - 读 tools/fish/configs/zh/{templates.toml, templates/*.png, config.yaml}
//   - 写 bin/data/containers/fishing-v2/templates/<key>/<WxH>.{png,json} + _meta.json
//   - patch bin/data/containers/fishing-v2/subgraphs/{state_FISHING,inspect_phase,try_hook_F}.json
//     的 ColorBarTrack 节点 config.roi → config.rois (来自 config.yaml bar_rois).
//
// 用法:
//   go run ./cmd/import-fish-data            # 实跑
//   go run ./cmd/import-fish-data --dry-run  # 不动文件, 只输出 plan
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"

	"yhbox/internal/services/container"
	templatepkg "yhbox/internal/services/template"
)

const (
	fishCfgDir = "tools/fish/configs/zh"
	ctnDir     = "bin/data/containers/fishing-v2"
)

type tomlConfig struct {
	Fish map[string][]tomlTemplate `toml:"fish"`
}
type tomlTemplate struct {
	File       string `toml:"file"`
	Resolution [2]int `toml:"resolution"`
	BBox       [4]int `toml:"bbox"`
	Note       string `toml:"note,omitempty"`
}

type ImportPlan struct {
	Templates       []TemplatePlan
	ColorBarPatches []ColorBarPatch
}

type TemplatePlan struct {
	Key      string
	KeyMeta  templatepkg.KeyMeta
	Variants []VariantPlan
}

type VariantPlan struct {
	Resolution [2]int
	BBox       [4]int   // first entry's bbox
	Regions    [][4]int // all entries' bbox (incl. first) when N > 1, else nil
	Note       string
	PngPath    string
}

type ColorBarPatch struct {
	SubgraphFile string
	NodeID       string
	NewRois      []map[string]interface{}
}

func (p *ImportPlan) totalVariants() int {
	n := 0
	for _, t := range p.Templates {
		n += len(t.Variants)
	}
	return n
}

func planImport(fishCfgDir, ctnDir string) (*ImportPlan, error) {
	tomlBytes, err := os.ReadFile(filepath.Join(fishCfgDir, "templates.toml"))
	if err != nil {
		return nil, fmt.Errorf("read templates.toml: %w", err)
	}
	var cfg tomlConfig
	if err := toml.Unmarshal(tomlBytes, &cfg); err != nil {
		return nil, fmt.Errorf("parse templates.toml: %w", err)
	}

	plan := &ImportPlan{}
	// sort slot names for stable output
	slotNames := make([]string, 0, len(cfg.Fish))
	for slot := range cfg.Fish {
		slotNames = append(slotNames, slot)
	}
	sort.Strings(slotNames)
	for _, slot := range slotNames {
		tplList := cfg.Fish[slot]
		tp := TemplatePlan{
			Key: "fishing." + slot,
			KeyMeta: templatepkg.KeyMeta{
				Name:        slot,
				Description: firstNoteOrEmpty(tplList),
				Tags:        []string{},
				Origin: templatepkg.TemplateOrigin{
					Kind:     "imported",
					SourceID: "tools/fish/configs/zh",
				},
			},
		}
		// Group by (file, resolution) — multi-bbox entries (e.g. bait_product 3×2 grid)
		// merge into single VariantPlan with Regions = all bboxes.
		type groupKey struct {
			File string
			Res  [2]int
		}
		groups := map[groupKey][]tomlTemplate{}
		groupOrder := []groupKey{}
		for _, t := range tplList {
			k := groupKey{File: t.File, Res: t.Resolution}
			if _, seen := groups[k]; !seen {
				groupOrder = append(groupOrder, k)
			}
			groups[k] = append(groups[k], t)
		}
		for _, gk := range groupOrder {
			entries := groups[gk]
			pngPath := filepath.Join(fishCfgDir, "templates", gk.File)
			if _, err := os.Stat(pngPath); err != nil {
				return nil, fmt.Errorf("PNG missing for %s: %s: %w", tp.Key, pngPath, err)
			}
			vp := VariantPlan{
				Resolution: gk.Res,
				BBox:       entries[0].BBox,
				Note:       entries[0].Note,
				PngPath:    pngPath,
			}
			if len(entries) > 1 {
				vp.Regions = make([][4]int, 0, len(entries))
				for _, e := range entries {
					vp.Regions = append(vp.Regions, e.BBox)
				}
			}
			tp.Variants = append(tp.Variants, vp)
		}
		plan.Templates = append(plan.Templates, tp)
	}

	patches, err := planColorBarPatches(fishCfgDir, ctnDir)
	if err != nil {
		return nil, err
	}
	plan.ColorBarPatches = patches

	return plan, nil
}

func firstNoteOrEmpty(tpls []tomlTemplate) string {
	for _, t := range tpls {
		if t.Note != "" {
			return t.Note
		}
	}
	return ""
}

type yamlConfig struct {
	Version int `yaml:"version"`
	Capture struct {
		BarROIs []struct {
			Resolution [2]int `yaml:"resolution"`
			BBox       [4]int `yaml:"bbox"`
		} `yaml:"bar_rois"`
	} `yaml:"capture"`
}

func planColorBarPatches(fishCfgDir, ctnDir string) ([]ColorBarPatch, error) {
	yamlBytes, err := os.ReadFile(filepath.Join(fishCfgDir, "config.yaml"))
	if err != nil {
		return nil, fmt.Errorf("read config.yaml: %w", err)
	}
	var ycfg yamlConfig
	if err := yaml.Unmarshal(yamlBytes, &ycfg); err != nil {
		return nil, fmt.Errorf("parse config.yaml: %w", err)
	}
	if len(ycfg.Capture.BarROIs) == 0 {
		return nil, fmt.Errorf("config.yaml::capture.bar_rois empty")
	}

	newRois := []map[string]interface{}{}
	for _, b := range ycfg.Capture.BarROIs {
		newRois = append(newRois, map[string]interface{}{
			"resolution": []int{b.Resolution[0], b.Resolution[1]},
			"x":          b.BBox[0],
			"y":          b.BBox[1],
			"w":          b.BBox[2] - b.BBox[0],
			"h":          b.BBox[3] - b.BBox[1],
		})
	}

	sgFiles := []string{"state_FISHING.json", "inspect_phase.json", "try_hook_F.json"}
	patches := []ColorBarPatch{}
	for _, fname := range sgFiles {
		path := filepath.Join(ctnDir, "subgraphs", fname)
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
		var sg struct {
			Graph struct {
				Nodes []struct {
					ID   string `json:"id"`
					Kind string `json:"kind"`
				} `json:"nodes"`
			} `json:"graph"`
		}
		if err := json.Unmarshal(data, &sg); err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
		for _, n := range sg.Graph.Nodes {
			if n.Kind == "ColorBarTrack" {
				patches = append(patches, ColorBarPatch{
					SubgraphFile: fname,
					NodeID:       n.ID,
					NewRois:      newRois,
				})
			}
		}
	}
	if len(patches) != 3 {
		return nil, fmt.Errorf("expected 3 ColorBarTrack nodes (1 per subgraph), got %d", len(patches))
	}
	return patches, nil
}

func executePlan(plan *ImportPlan) error {
	templatesDir := filepath.Join(ctnDir, "templates")

	// clean old templates before re-import
	if err := os.RemoveAll(templatesDir); err != nil {
		return fmt.Errorf("rm old templates: %w", err)
	}

	st, err := templatepkg.NewStore(templatesDir)
	if err != nil {
		return fmt.Errorf("create store: %w", err)
	}
	for _, tp := range plan.Templates {
		if err := st.SaveMeta(tp.Key, tp.KeyMeta); err != nil {
			return fmt.Errorf("SaveMeta %s: %w", tp.Key, err)
		}
		for _, v := range tp.Variants {
			pngData, err := os.ReadFile(v.PngPath)
			if err != nil {
				return fmt.Errorf("read PNG %s: %w", v.PngPath, err)
			}
			_, err = st.SaveVariant(tp.Key, pngData, templatepkg.VariantMeta{
				Resolution: v.Resolution,
				BBox:       v.BBox,
				Regions:    v.Regions,
				Note:       v.Note,
			})
			if err != nil {
				return fmt.Errorf("SaveVariant %s @%dx%d: %w", tp.Key, v.Resolution[0], v.Resolution[1], err)
			}
		}
	}
	fmt.Printf("OK Wrote %d keys, %d variants\n", len(plan.Templates), plan.totalVariants())

	// patch 3 ColorBarTrack node configs
	for _, p := range plan.ColorBarPatches {
		path := filepath.Join(ctnDir, "subgraphs", p.SubgraphFile)
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var sg map[string]interface{}
		if err := json.Unmarshal(data, &sg); err != nil {
			return err
		}
		graph, ok := sg["graph"].(map[string]interface{})
		if !ok {
			return fmt.Errorf("%s: graph not object", p.SubgraphFile)
		}
		nodes, ok := graph["nodes"].([]interface{})
		if !ok {
			return fmt.Errorf("%s: graph.nodes not array", p.SubgraphFile)
		}
		patched := false
		for i, ni := range nodes {
			n, _ := ni.(map[string]interface{})
			if n["id"] == p.NodeID && n["kind"] == "ColorBarTrack" {
				config, _ := n["config"].(map[string]interface{})
				if config == nil {
					config = map[string]interface{}{}
				}
				delete(config, "roi")
				config["rois"] = p.NewRois
				n["config"] = config
				nodes[i] = n
				patched = true
				break
			}
		}
		if !patched {
			return fmt.Errorf("%s: ColorBarTrack node %q not found", p.SubgraphFile, p.NodeID)
		}
		graph["nodes"] = nodes
		sg["graph"] = graph
		out, err := json.MarshalIndent(sg, "", "  ")
		if err != nil {
			return err
		}
		if err := os.WriteFile(path, out, 0o644); err != nil {
			return err
		}
		fmt.Printf("OK Patched %s (ColorBarTrack@%s)\n", p.SubgraphFile, p.NodeID)
	}
	return nil
}

func runValidator(ctnDir string) error {
	containerPath := filepath.Join(ctnDir, "container.json")
	data, err := os.ReadFile(containerPath)
	if err != nil {
		return fmt.Errorf("read container.json: %w", err)
	}
	var c container.Container
	if err := json.Unmarshal(data, &c); err != nil {
		return fmt.Errorf("parse container.json: %w", err)
	}

	sgDir := filepath.Join(ctnDir, "subgraphs")
	entries, _ := os.ReadDir(sgDir)
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		sgData, _ := os.ReadFile(filepath.Join(sgDir, e.Name()))
		var sg container.Subgraph
		if err := json.Unmarshal(sgData, &sg); err != nil {
			return fmt.Errorf("parse %s: %w", e.Name(), err)
		}
		c.Subgraphs = append(c.Subgraphs, sg)
	}

	errs := container.ValidateContainer(&c)
	hasErr := false
	for _, e := range errs {
		if e.Severity == container.SeverityError {
			hasErr = true
		}
		fmt.Fprintf(os.Stderr, "[validator] [%s] %s @ %s — %v\n", e.Severity, e.Code, e.NodeID, e.Params)
	}
	if hasErr {
		return fmt.Errorf("validation failed")
	}
	return nil
}

func printDryRunSummary(plan *ImportPlan) {
	fmt.Printf("[dry-run] would write %d template keys × variants:\n", len(plan.Templates))
	for _, tp := range plan.Templates {
		variantStrs := []string{}
		for _, v := range tp.Variants {
			extra := ""
			if len(v.Regions) > 1 {
				extra = fmt.Sprintf("[%d regions]", len(v.Regions))
			}
			variantStrs = append(variantStrs, fmt.Sprintf("%dx%d%s", v.Resolution[0], v.Resolution[1], extra))
		}
		fmt.Printf("  - %s/  (%s | _meta.json)\n", tp.Key, strings.Join(variantStrs, " | "))
	}
	fmt.Printf("[dry-run] would patch %d ColorBarTrack node configs in subgraphs:\n", len(plan.ColorBarPatches))
	for _, p := range plan.ColorBarPatches {
		fmt.Printf("  - %s::node[%s] roi → rois[%d]\n", p.SubgraphFile, p.NodeID, len(p.NewRois))
	}
	templatesDir := filepath.Join(ctnDir, "templates")
	if entries, err := os.ReadDir(templatesDir); err == nil {
		count := 0
		for _, e := range entries {
			if e.IsDir() {
				if vs, _ := os.ReadDir(filepath.Join(templatesDir, e.Name())); vs != nil {
					count += len(vs)
				}
			} else {
				count++
			}
		}
		fmt.Printf("[dry-run] would os.RemoveAll %s (current contents: %d files/subdirs)\n", templatesDir, count)
	}
}

func main() {
	dryRun := flag.Bool("dry-run", false, "show what would be written without touching files")
	flag.Parse()

	plan, err := planImport(fishCfgDir, ctnDir)
	if err != nil {
		fatal(err)
	}

	if *dryRun {
		printDryRunSummary(plan)
		fmt.Println("OK Dry-run complete. No files touched.")
		return
	}

	if err := executePlan(plan); err != nil {
		fatal(err)
	}
	if err := runValidator(ctnDir); err != nil {
		fatal(err)
	}

	fmt.Printf("OK Import done. %d keys × variants = %d PNG. %d ColorBarTrack nodes patched. Validator clean.\n",
		len(plan.Templates), plan.totalVariants(), len(plan.ColorBarPatches))
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "ERR %v\n", err)
	os.Exit(1)
}
