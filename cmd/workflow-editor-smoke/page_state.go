package main

type pageState struct {
	Href                  string         `json:"href"`
	NodeAddTrigger        bool           `json:"nodeAddTrigger"`
	WorkspaceTools        int            `json:"workspaceTools"`
	GraphManager          bool           `json:"graphManager"`
	CanvasNodes           int            `json:"canvasNodes"`
	CanvasEdges           int            `json:"canvasEdges"`
	AIReview              bool           `json:"aiReview"`
	WorkflowState         bool           `json:"workflowState"`
	ResourceDock          bool           `json:"resourceDock"`
	ResourceKind          string         `json:"resourceKind"`
	ResourceCreate        bool           `json:"resourceCreate"`
	ResourceScope         string         `json:"resourceScope"`
	ResourceScopeActive   int            `json:"resourceScopeActive"`
	ResourceScopeContrast bool           `json:"resourceScopeContrast"`
	ResourceModeControls  int            `json:"resourceModeControls"`
	ResourceFiltersFill   bool           `json:"resourceFiltersFill"`
	ResourceLoading       bool           `json:"resourceLoading"`
	RecipeItems           int            `json:"recipeItems"`
	SnippetDock           bool           `json:"snippetDock"`
	SnippetItems          int            `json:"snippetItems"`
	SnippetModal          bool           `json:"snippetModal"`
	NodeContextMenu       bool           `json:"nodeContextMenu"`
	TemplateMenuActions   int            `json:"templateMenuActions"`
	RunStarted            bool           `json:"runStarted"`
	AssetsView            bool           `json:"assetsView"`
	AssetsRecording       bool           `json:"assetsRecording"`
	AssetBrowse           bool           `json:"assetBrowse"`
	AssetManageButton     bool           `json:"assetManageButton"`
	AssetManagement       bool           `json:"assetManagement"`
	SchedulesView         bool           `json:"schedulesView"`
	ScheduleBrowse        bool           `json:"scheduleBrowse"`
	ScheduleManageButton  bool           `json:"scheduleManageButton"`
	ScheduleManagement    bool           `json:"scheduleManagement"`
	ScheduleEditor        bool           `json:"scheduleEditor"`
	ScheduleAdvanced      bool           `json:"scheduleAdvanced"`
	ScheduleAdvToggle     bool           `json:"scheduleAdvancedToggle"`
	ScheduleInterval      bool           `json:"scheduleTargetInterval"`
	ScheduleRows          int            `json:"scheduleRows"`
	ScheduleRowTargets    []string       `json:"scheduleRowTargets"`
	ScheduleRowStatuses   []string       `json:"scheduleRowStatuses"`
	ScheduleEditTargets   []string       `json:"scheduleEditTargets"`
	SettingsView          bool           `json:"settingsView"`
	SettingsGroups        int            `json:"settingsGroups"`
	AppContextTitle       string         `json:"appContextTitle"`
	CreateInput           bool           `json:"createInput"`
	RecoveryPanel         bool           `json:"recoveryPanel"`
	WorkflowBrowse        bool           `json:"workflowBrowse"`
	WorkflowManageButton  bool           `json:"workflowManageButton"`
	WorkflowManagement    bool           `json:"workflowManagement"`
	WorkflowRows          int            `json:"workflowRows"`
	WorkflowTotal         int            `json:"workflowTotal"`
	LauncherButton        bool           `json:"launcherButton"`
	GraphChromeDark       bool           `json:"graphChromeDark"`
	HandleOverlaps        int            `json:"handleOverlaps"`
	NativeConfirmCalls    int            `json:"nativeConfirmCalls"`
	ConfirmDialog         bool           `json:"confirmDialog"`
	Dirty                 bool           `json:"dirty"`
	SaveInlineFeedback    bool           `json:"saveInlineFeedback"`
	SaveError             string         `json:"saveError"`
	SaveToast             bool           `json:"saveToast"`
	Diagnostics           bool           `json:"diagnostics"`
	MissingInputWarnings  int            `json:"missingInputWarnings"`
	SelectedNodes         int            `json:"selectedNodes"`
	SelectionToolbar      bool           `json:"selectionToolbar"`
	ConnectionMenu        bool           `json:"connectionMenu"`
	ConnectionCandidates  int            `json:"connectionCandidates"`
	ConnectionError       string         `json:"connectionError"`
	Debugger              bool           `json:"debugger"`
	DebugStart            bool           `json:"debugStart"`
	DebugPaused           bool           `json:"debugPaused"`
	DebugBusy             bool           `json:"debugBusy"`
	DebugCompleted        bool           `json:"debugCompleted"`
	DebugCurrent          int            `json:"debugCurrent"`
	DebugNode             string         `json:"debugNode"`
	Breakpoints           int            `json:"breakpoints"`
	CurrentGraph          string         `json:"currentGraph"`
	GraphCalls            int            `json:"graphCalls"`
	GraphBoundaries       int            `json:"graphBoundaries"`
	GraphInterface        bool           `json:"graphInterface"`
	BoundaryClipped       int            `json:"boundaryClipped"`
	BoundaryObscured      int            `json:"boundaryObscured"`
	MinimapToggle         bool           `json:"minimapToggle"`
	MinimapOpen           bool           `json:"minimapOpen"`
	Annotations           int            `json:"annotations"`
	GraphNameInput        bool           `json:"graphNameInput"`
	CallMenuOptions       int            `json:"callMenuOptions"`
	Reroutes              int            `json:"reroutes"`
	NodeOverlaps          int            `json:"nodeOverlaps"`
	NodeGeometry          []nodeGeometry `json:"nodeGeometry"`
	Errors                []string       `json:"errors"`
}

type canvasNodeErgonomics struct {
	CenterX                float64 `json:"centerX"`
	CenterY                float64 `json:"centerY"`
	BlankX                 float64 `json:"blankX"`
	BlankY                 float64 `json:"blankY"`
	Width                  float64 `json:"width"`
	Height                 float64 `json:"height"`
	Zoom                   float64 `json:"zoom"`
	Selected               bool    `json:"selected"`
	CompositeInlineEditors int     `json:"compositeInlineEditors"`
}

type wheelOwnershipProbe struct {
	X    float64 `json:"x"`
	Y    float64 `json:"y"`
	Zoom float64 `json:"zoom"`
}

type nodeGeometry struct {
	ID        string  `json:"id"`
	X         float64 `json:"x"`
	Y         float64 `json:"y"`
	Width     float64 `json:"width"`
	Height    float64 `json:"height"`
	Transform string  `json:"transform"`
}

type point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type connectionGesture struct {
	Start point `json:"start"`
	End   point `json:"end"`
}
