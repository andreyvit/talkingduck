# macOS settings UI

Use macOS conventions where Wails and WebKit support them cleanly. Match
behavior, layout, and typography before adding decorative effects. These are
TalkingDuck design rules, not a claim that HTML controls equal AppKit controls.

## Window and navigation

- Use one reusable Settings window, opened by the app menu's Settings… and ⌘,.
  Closing it hides it; it does not quit the app. Keep normal text-editing menus.
- Keep the title bar native. Do not draw traffic lights, fake toolbars, or a
  second large page title beneath the window title.
- Group settings by purpose: Applications, Tickets, Folders, Connection, Configuration File.
  Avoid a sidebar while these fit in one understandable page.
- Scroll the entire page. No fixed-height folder list or nested scroll area.
  Native popup menus may scroll their options; that is a different control.
- Allow resizing because paths and folder counts vary. This deliberately
  relaxes the fixed-size settings-window convention. No horizontal scrolling.

## Alignment and spacing

The form grid is the main source of the Mac feel. Never position each row
independently or repair alignment with spaces in labels.

- Right-align form labels to one axis; start all controls at another. Use the
  same grid for editor, Git client, tracker, custom fields, and connection.
- Center labels against a single-line control's text. For multiline controls,
  align the label to the first line. Align like controls to like controls.
- Keep related controls the same width. Path field + Choose… occupy one control
  column together. Help and validation begin at that column.
- Use shared tokens: 24 px page inset/section separation, 12 px group inset,
  16 px label/control gap, 10 px between form rows, 8 px before help. These are
  project starting values in CSS pixels, not Apple-mandated measurements.
- Use a 156 px label column and a flexible control column. Keep the page's
  maximum width bounded; extra window width must not scatter controls.
- Folder headers and controls share fixed column boundaries. Use comes first
  as a checkbox; the path follows, then the menu bar shortcut. Long paths wrap
  within the folder column. Dropdown positions must not depend on name length.
- Judge optical alignment at actual size. Check text baselines, control centers,
  group edges, separator insets, label/control gaps, and clipped popup arrows.
  A mathematically centered box can still look wrong because of native padding.

## Controls and typography

- Use the system font, regular 13 px labels, restrained 13 px semibold section
  headings, and 11 px secondary text. No oversized web headings or all-caps labels.
- Prefer native HTML selects, buttons, text fields, and native file panels.
  Preserve their keyboard handling and focus rings. Do not custom-draw menus.
- Labels remain visible; placeholders are examples, never the only label.
  Use real label/control associations and accessible status messages.
- Use Choose… for an operation that opens a panel. Use concise nouns for settings
  and verbs for actions. Show custom fields only when Custom is selected.
- Follow light/dark appearance with restrained semantic foreground, secondary,
  background, separator, and error colors. Do not use color as the only feedback.
- Keep help beneath the related control, aligned with its leading edge. Avoid
  centering help across the entire group or letting errors shift other columns.

## Editing behavior

- Save on change with a short cooldown and throttle for text forms. No Apply,
  Saved, or other success messages. Keep errors visible and preserve working
  settings until a valid draft can replace them.
- Validate before saving, preserve invalid text, explain the field to fix,
  and keep the last working configuration. Show errors beside that form.
- Do not replace focused controls or overwrite unfinished edits when snapshots
  arrive. Ignore stale save acknowledgments if the user has edited again.
- Tab order follows the page. Command-A/C/V/Z work in text fields. Inventory
  single-letter commands never run while entering settings text.
- Escape dismisses transient menus/panels; it does not discard the Settings
  window. Command-W closes the window. Keep native window/menu behavior where
  available rather than inventing key handling in every form.

## Review and practical limits

Inspect the built app at normal size in light and dark appearances, minimum and
normal window sizes, with a long path, Custom fields, URL errors, and an open
popup. Scroll from top to bottom: only one document scroller. Test mouse and
keyboard edits while background snapshots arrive. Check focus and selection
survive, not merely that a screenshot looks right.

Wails provides a native window/file panel around WebKit content. We can reproduce
form grouping, alignment, typography, editing conventions, and accessibility
labels. AppKit vibrancy, exact OS-version control metrics, native Form layout,
Window-menu integration, and all accessibility settings may need native code.
Do not add fragile private APIs or a bespoke control toolkit to imitate these.
Record an explicit compromise instead. Behavior and readability outrank effects.

## Sources and examples

- [Apple: Settings](https://developer.apple.com/design/human-interface-guidelines/settings)
  and [adding a settings interface](https://developer.apple.com/documentation/foundation/adding-a-settings-interface-to-your-app)
  establish app-level settings and the macOS Settings scene/window pattern.
- [Human Interface Guidelines Extras](https://github.com/sindresorhus/human-interface-guidelines-extras)
  recommends grouped forms, preserving useful standard menus, and treating
  Settings as a regular window. Community guidance, not official Apple policy.
- [Watts Martin: What Makes an App Feel Right on the Mac?](https://coyotetracks.org/blog/app-feel-on-mac/)
  discusses menu, shortcut, window, and settings conventions. Nova, MarsEdit,
  and Pages are positive examples; framework choice alone does not decide fit.
- [John Gruber: app feel](https://daringfireball.net/linked/2025/04/02/apps-feel-right-mac-watts-martin)
  contrasts a proper settings interface with editor tabs/config files and
  unconventional cross-platform controls.
- [Gruber: Ventura System Settings fit and finish](https://daringfireball.net/linked/2022/08/15/ventura-system-settings-tonsky)
  discusses Tonsky's visual examples: broken alignment and clipped controls
  undermine a settings UI independently of its overall navigation model.

Sources reviewed 2026-10-06. The measurements above are our implementation
choices; the criticisms are historical examples, not claims about current macOS.
