# Spec structure

Keep the inventory design in one spec file until the user requests a split.

- At most 7 sections (`##`) per spec.
- At most 7 subsections (`###`) per section.
- At most 7 bullet items per subsection.
- At most 7 nested items per bullet item.
- Prefer four structural levels: section, subsection, item, sub-item.
- Use a fifth level only when necessary; explain why in the spec.
- Keep observed behavior, inferred rules, and open decisions distinct. Update the
  spec with implementation changes; links may point to supporting research.

A document title and frontmatter do not count as structural levels. Do not use
extra headings or long unstructured lists to evade these limits. This is a living
product/behavior spec; creating it does not activate the Zoo execution workflow.
