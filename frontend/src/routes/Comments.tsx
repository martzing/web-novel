import { useState } from "react";
import { Link, useParams } from "react-router-dom";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { api, type Comment } from "../lib/api";
import { useAuth } from "../lib/auth";
import { numberTH, relativeTime } from "../lib/format";
import { Empty, ErrorNote, Loading, Tabs } from "../components";

type Sort = "popular" | "latest" | "with_replies";

export default function Comments() {
  const { id = "" } = useParams();
  const { user } = useAuth();
  const qc = useQueryClient();

  const [sort, setSort] = useState<Sort>("popular");
  const [body, setBody] = useState("");
  const [spoiler, setSpoiler] = useState(false);
  const [replyTo, setReplyTo] = useState<string | null>(null);

  const chapter = useQuery({ queryKey: ["chapter", id], queryFn: () => api.getChapter(id) });
  const comments = useQuery({
    queryKey: ["comments", id, sort],
    queryFn: () => api.listComments(id, sort),
  });

  const post = useMutation({
    mutationFn: () =>
      api.createComment(id, {
        body,
        parent_id: replyTo ?? undefined,
        is_spoiler_hidden: spoiler,
      }),
    onSuccess: () => {
      setBody("");
      setSpoiler(false);
      setReplyTo(null);
      qc.invalidateQueries({ queryKey: ["comments", id] });
    },
  });

  const toggleLike = useMutation({
    mutationFn: ({ commentId, liked }: { commentId: string; liked: boolean }) =>
      liked ? api.unlikeComment(commentId) : api.likeComment(commentId),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["comments", id] }),
  });

  const remove = useMutation({
    mutationFn: (commentId: string) => api.deleteComment(commentId),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["comments", id] }),
  });

  const rows = comments.data?.data ?? [];

  return (
    <section>
      {chapter.data && (
        <Link to={`/read/${id}`} className="muted" style={{ fontSize: 12.5 }}>
          ← {chapter.data.novel_title_th}
        </Link>
      )}

      <div className="page-head" style={{ marginTop: 14 }}>
        <h1 className="page-title">ความเห็นบทที่ {chapter.data?.chapter_no ?? ""}</h1>
      </div>
      {chapter.data && (
        <div className="muted" style={{ fontSize: 12.5, marginTop: 6 }}>
          {chapter.data.title}
        </div>
      )}

      {user ? (
        <div className="card" style={{ marginTop: 22 }}>
          {replyTo && (
            <div className="muted" style={{ fontSize: 12, marginBottom: 8 }}>
              กำลังตอบกลับ ·{" "}
              <button className="btn btn--ghost btn--sm" onClick={() => setReplyTo(null)}>
                ยกเลิก
              </button>
            </div>
          )}
          <textarea
            className="textarea"
            value={body}
            onChange={(e) => setBody(e.target.value)}
            placeholder="แบ่งปันความคิดเห็นเกี่ยวกับบทนี้"
          />
          {post.isError && <div className="form-error">{(post.error as Error).message}</div>}
          <div
            style={{
              display: "flex",
              justifyContent: "space-between",
              alignItems: "center",
              marginTop: 12,
              gap: 12,
              flexWrap: "wrap",
            }}
          >
            <label style={{ display: "flex", alignItems: "center", gap: 8, fontSize: 12.5 }}>
              <input type="checkbox" checked={spoiler} onChange={(e) => setSpoiler(e.target.checked)} />
              ซ่อนสปอยล์
            </label>
            <button
              className="btn btn--primary"
              disabled={body.trim() === "" || post.isPending}
              onClick={() => post.mutate()}
            >
              ส่งความเห็น
            </button>
          </div>
        </div>
      ) : (
        <Empty>
          <Link to="/login">เข้าสู่ระบบ</Link> เพื่อร่วมแสดงความเห็น
        </Empty>
      )}

      <div style={{ marginTop: 26 }}>
        <Tabs<Sort>
          active={sort}
          onChange={setSort}
          tabs={[
            { key: "popular", label: "ยอดนิยม" },
            { key: "latest", label: "ล่าสุด" },
            { key: "with_replies", label: "เฉพาะที่มีคนตอบ" },
          ]}
        />
      </div>

      {comments.isError ? (
        <ErrorNote message={(comments.error as Error).message} />
      ) : comments.isLoading ? (
        <Loading />
      ) : rows.length === 0 ? (
        <Empty>ยังไม่มีความเห็นในบทนี้ เป็นคนแรกได้เลย</Empty>
      ) : (
        <div className="grid" style={{ marginTop: 18 }}>
          {rows.map((comment) => (
            <CommentCard
              key={comment.id}
              comment={comment}
              canModerate={Boolean(user)}
              onReply={() => setReplyTo(comment.id)}
              onLike={() => toggleLike.mutate({ commentId: comment.id, liked: comment.liked })}
              onDelete={() => remove.mutate(comment.id)}
              onLikeReply={(reply) => toggleLike.mutate({ commentId: reply.id, liked: reply.liked })}
            />
          ))}
        </div>
      )}
    </section>
  );
}

function CommentCard({
  comment,
  canModerate,
  onReply,
  onLike,
  onDelete,
  onLikeReply,
}: {
  comment: Comment;
  canModerate: boolean;
  onReply: () => void;
  onLike: () => void;
  onDelete: () => void;
  onLikeReply: (reply: Comment) => void;
}) {
  return (
    <div className="card">
      <CommentBody comment={comment} />

      <div style={{ display: "flex", gap: 14, marginTop: 12, alignItems: "center" }}>
        <button
          className="btn btn--ghost btn--sm"
          onClick={onLike}
          style={{ color: comment.liked ? "var(--red)" : undefined }}
        >
          เห็นด้วย {numberTH(comment.likes_count)}
        </button>
        {canModerate && (
          <>
            <button className="btn btn--ghost btn--sm" onClick={onReply}>
              ตอบกลับ
            </button>
            <button className="btn btn--ghost btn--sm" onClick={onDelete}>
              ลบ
            </button>
          </>
        )}
      </div>

      {comment.replies.length > 0 && (
        <div
          style={{
            marginTop: 14,
            paddingTop: 14,
            paddingLeft: 14,
            borderTop: "1px solid var(--line)",
            borderLeft: "2px solid var(--line)",
            display: "grid",
            gap: 14,
          }}
        >
          {comment.replies.map((reply) => (
            <div key={reply.id}>
              <CommentBody comment={reply} />
              <button
                className="btn btn--ghost btn--sm"
                style={{ marginTop: 8, color: reply.liked ? "var(--red)" : undefined }}
                onClick={() => onLikeReply(reply)}
              >
                เห็นด้วย {numberTH(reply.likes_count)}
              </button>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

function CommentBody({ comment }: { comment: Comment }) {
  const [revealed, setRevealed] = useState(false);
  const hidden = comment.is_spoiler_hidden && !revealed;

  return (
    <>
      <div style={{ display: "flex", alignItems: "center", gap: 10, flexWrap: "wrap" }}>
        <span style={{ fontSize: 13.5 }}>{comment.author.display_name}</span>
        {comment.is_translator && <span className="pill pill--red">ผู้แปล</span>}
        {comment.author.role === "admin" && <span className="pill pill--gold">ผู้ดูแล</span>}
        <span className="muted" style={{ fontSize: 11.5 }}>
          {relativeTime(comment.created_at)}
        </span>
      </div>

      {hidden ? (
        <button
          className="btn btn--sm"
          style={{ marginTop: 10 }}
          onClick={() => setRevealed(true)}
        >
          ความเห็นนี้มีสปอยล์ · แตะเพื่อแสดง
        </button>
      ) : (
        <p style={{ fontSize: 13.5, lineHeight: 1.95, marginTop: 8, marginBottom: 0 }}>
          {comment.body}
        </p>
      )}
    </>
  );
}
