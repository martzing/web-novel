/**
 * Barrel for the shared UI kit.
 *
 * Re-exports only — every component lives in its own file so that importing
 * `Modal` does not drag the rest of the kit into the bundle graph with it.
 */
export { COVER_COLORS, COVER_STYLES, Cover, NovelCover } from "./Cover";
export type { CoverFields, CoverProps } from "./Cover";
export { Empty, ErrorNote, Loading } from "./Feedback";
export { Modal } from "./Modal";
export { ProgressBar } from "./ProgressBar";
export { ShelfRow } from "./ShelfRow";
export { StarPicker, Stars } from "./Stars";
export { Tabs } from "./Tabs";
