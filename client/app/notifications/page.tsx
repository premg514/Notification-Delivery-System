"use client";

import Link from "next/link";
import { useEffect, useMemo, useState } from "react";
import { getRecentNotifications, type RecentNotification } from "@/lib/api";

const PAGE_SIZE = 10;
const MAX_FETCH_LIMIT = 1000;

function getStatusClasses(status: string): string {
  switch (status) {
    case "process":
      return "bg-sky-50 text-sky-700";
    case "queue":
      return "bg-amber-50 text-amber-700";
    case "sent":
      return "bg-emerald-50 text-emerald-700";
    default:
      return "bg-slate-100 text-slate-700";
  }
}

function formatDateTime(value: string): string {
  return new Intl.DateTimeFormat("en-IN", {
    dateStyle: "medium",
    timeStyle: "medium",
    hour12: false,
  }).format(new Date(value));
}

export default function NotificationsPage(): JSX.Element {
  const [notifications, setNotifications] = useState<RecentNotification[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);
  const [searchTerm, setSearchTerm] = useState<string>("");
  const [currentPage, setCurrentPage] = useState<number>(1);

  useEffect((): (() => void) => {
    let mounted = true;

    const loadNotifications = async (): Promise<void> => {
      try {
        const result = await getRecentNotifications(MAX_FETCH_LIMIT);
        if (!mounted) {
          return;
        }
        setNotifications(result);
      } catch {
        if (!mounted) {
          return;
        }
        setError("Failed to load notifications.");
      } finally {
        if (mounted) {
          setLoading(false);
        }
      }
    };

    void loadNotifications();

    return (): void => {
      mounted = false;
    };
  }, []);

  const filteredNotifications = useMemo(() => {
    const query = searchTerm.trim().toLowerCase();
    if (query === "") {
      return notifications;
    }

    return notifications.filter((notification) =>
      [
        notification.id,
        notification.title,
        notification.target_department,
        notification.priority,
        notification.status,
      ]
        .join(" ")
        .toLowerCase()
        .includes(query),
    );
  }, [notifications, searchTerm]);

  const totalPages = Math.max(1, Math.ceil(filteredNotifications.length / PAGE_SIZE));

  const paginatedRows = useMemo(() => {
    const start = (currentPage - 1) * PAGE_SIZE;
    return filteredNotifications.slice(start, start + PAGE_SIZE);
  }, [currentPage, filteredNotifications]);

  useEffect(() => {
    setCurrentPage(1);
  }, [searchTerm]);

  useEffect(() => {
    if (currentPage > totalPages) {
      setCurrentPage(totalPages);
    }
  }, [currentPage, totalPages]);

  return (
    <main className="min-h-screen px-3 py-4 sm:px-6 lg:px-10">
      <div className="mx-auto max-w-7xl panel rounded-[30px] px-5 py-5 shadow-panel sm:px-8">
        <div className="mb-6 flex flex-wrap items-center justify-between gap-4">
          <div>
            <p className="text-xs font-semibold uppercase tracking-[0.25em] text-slate-400">
              Notification history
            </p>
            <h1 className="font-display text-3xl text-ink sm:text-4xl">View All Notifications</h1>
          </div>
          <Link
            href="/"
            className="inline-flex items-center rounded-full bg-slate-100 px-4 py-2 text-sm font-bold text-slate-600 transition hover:bg-slate-200"
          >
            Back to dashboard
          </Link>
        </div>

        <div className="mb-5">
          <label className="block">
            <span className="mb-2 block text-xs font-semibold uppercase tracking-[0.25em] text-slate-400">
              Search notifications
            </span>
            <input
              value={searchTerm}
              onChange={(event) => setSearchTerm(event.target.value)}
              placeholder="Search by id, title, department, priority, or status"
              className="w-full rounded-2xl border border-slate-200 bg-white px-4 py-3 text-sm text-ink outline-none transition focus:border-coral"
            />
          </label>
        </div>

        <div className="space-y-3 md:hidden">
          {paginatedRows.map((notification) => (
            <article
              key={notification.id}
              className="rounded-2xl border border-slate-100 bg-white/70 p-4"
            >
              <div className="mb-2 flex items-center justify-between gap-3">
                <span className="rounded-full bg-slate-100 px-3 py-1 text-xs font-bold uppercase text-slate-600">
                  {notification.target_department}
                </span>
                <span className={`rounded-full px-3 py-1 text-xs font-bold uppercase ${getStatusClasses(notification.status)}`}>
                  {notification.status}
                </span>
              </div>
              <p className="text-base font-black text-ink break-words">{notification.title}</p>
              <div className="mt-2 text-xs uppercase tracking-wide text-slate-500">
                <p>Priority: {notification.priority}</p>
                <p className="mt-1">{formatDateTime(notification.created_at)}</p>
              </div>
            </article>
          ))}
          {!loading && !error && paginatedRows.length === 0 ? (
            <div className="rounded-2xl border border-slate-100 bg-white/70 p-6 text-center text-sm text-slate-400">
              {searchTerm.trim()
                ? "No notifications match your search."
                : "No notifications available."}
            </div>
          ) : null}
        </div>

        <div className="hidden overflow-x-auto md:block">
          <table className="min-w-[560px] w-full border-separate border-spacing-0 text-left">
            <thead>
              <tr className="text-xs font-semibold uppercase tracking-[0.25em] text-slate-400">
                <th className="hidden px-4 py-4 sm:table-cell">Created At</th>
                <th className="px-4 py-4">Title</th>
                <th className="px-4 py-4">Department</th>
                <th className="hidden px-4 py-4 md:table-cell">Priority</th>
                <th className="px-4 py-4">Status</th>
              </tr>
            </thead>
            <tbody>
              {paginatedRows.map((notification) => (
                <tr key={notification.id} className="border-t border-slate-100 text-base text-ink">
                  <td className="hidden px-4 py-4 text-sm text-slate-500 sm:table-cell">
                    {formatDateTime(notification.created_at)}
                  </td>
                  <td className="px-4 py-4 font-semibold break-words">{notification.title}</td>
                  <td className="px-4 py-4">
                    <span className="rounded-full bg-slate-100 px-3 py-1 text-xs font-bold uppercase text-slate-600">
                      {notification.target_department}
                    </span>
                  </td>
                  <td className="hidden px-4 py-4 uppercase md:table-cell">{notification.priority}</td>
                  <td className="px-4 py-4">
                    <span className={`rounded-full px-3 py-1 text-xs font-bold uppercase ${getStatusClasses(notification.status)}`}>
                      {notification.status}
                    </span>
                  </td>
                </tr>
              ))}
              {!loading && !error && paginatedRows.length === 0 ? (
                <tr>
                  <td colSpan={5} className="px-4 py-10 text-center text-sm text-slate-400">
                    {searchTerm.trim()
                      ? "No notifications match your search."
                      : "No notifications available."}
                  </td>
                </tr>
              ) : null}
            </tbody>
          </table>
        </div>

        {loading ? <p className="mt-5 text-sm text-slate-500">Loading notifications...</p> : null}
        {error ? <p className="mt-5 text-sm font-semibold text-rose-600">{error}</p> : null}

        <div className="mt-6 flex flex-wrap items-center justify-between gap-3">
          <p className="text-sm text-slate-500">
            {filteredNotifications.length} result{filteredNotifications.length === 1 ? "" : "s"} | Page {currentPage} of {totalPages}
          </p>
          <div className="flex items-center gap-2">
            <button
              type="button"
              onClick={() => setCurrentPage((page) => Math.max(1, page - 1))}
              disabled={currentPage === 1}
              className="rounded-xl border border-slate-200 px-4 py-2 text-sm font-semibold text-slate-600 disabled:cursor-not-allowed disabled:opacity-50"
            >
              Previous
            </button>
            <button
              type="button"
              onClick={() => setCurrentPage((page) => Math.min(totalPages, page + 1))}
              disabled={currentPage >= totalPages}
              className="rounded-xl border border-slate-200 px-4 py-2 text-sm font-semibold text-slate-600 disabled:cursor-not-allowed disabled:opacity-50"
            >
              Next
            </button>
          </div>
        </div>
      </div>
    </main>
  );
}
