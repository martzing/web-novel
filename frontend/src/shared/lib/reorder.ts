import { useCallback, useState } from "react";

/** Moves one item to a new index, returning a new array. */
export function moveItem<T>(items: T[], from: number, to: number): T[] {
  if (from === to || from < 0 || to < 0 || from >= items.length || to >= items.length) {
    return items;
  }
  const next = items.slice();
  const [moved] = next.splice(from, 1);
  next.splice(to, 0, moved);
  return next;
}

export interface DragHandlers {
  draggable: true;
  onDragStart: (e: React.DragEvent) => void;
  onDragOver: (e: React.DragEvent) => void;
  onDrop: (e: React.DragEvent) => void;
  onDragEnd: () => void;
}

/**
 * Native HTML5 drag reordering.
 *
 * No dependency: the browser already has this, and a drag library would be a
 * large addition for one interaction. Drag is a pointer-only affordance
 * though — it is unusable with a keyboard and awkward on touch — so every
 * caller pairs it with the move up/down buttons this hook also serves.
 */
export function useReorder(onReorder: (from: number, to: number) => void) {
  const [dragging, setDragging] = useState<number | null>(null);
  const [over, setOver] = useState<number | null>(null);

  const handlersFor = useCallback(
    (index: number): DragHandlers => ({
      draggable: true,
      onDragStart: (e) => {
        setDragging(index);
        e.dataTransfer.effectAllowed = "move";
        // Firefox ignores a drag that carries no data.
        e.dataTransfer.setData("text/plain", String(index));
      },
      onDragOver: (e) => {
        e.preventDefault();
        e.dataTransfer.dropEffect = "move";
        setOver(index);
      },
      onDrop: (e) => {
        e.preventDefault();
        const from = dragging ?? Number(e.dataTransfer.getData("text/plain"));
        if (!Number.isNaN(from)) onReorder(from, index);
        setDragging(null);
        setOver(null);
      },
      onDragEnd: () => {
        setDragging(null);
        setOver(null);
      },
    }),
    [dragging, onReorder],
  );

  const classFor = useCallback(
    (index: number) => {
      if (dragging === index) return " is-dragging";
      if (over === index && dragging !== null) return " is-drop-target";
      return "";
    },
    [dragging, over],
  );

  return { handlersFor, classFor };
}
