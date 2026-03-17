"use client";

import Link from "next/link";
import { useEffect, useMemo, useState, type FormEvent } from "react";
import {
  getHealth,
  getRecentNotifications,
  sendNotification,
  type NotificationResponse,
  type RecentNotification,
} from "@/lib/api";

type Priority = "high" | "normal" | "low";
type Department = "CSE" | "ECE" | "ME" | "CIVIL" | "EEE";
type HealthState = "healthy" | "degraded" | "offline";
type ActivityStatus = "sent" | "duplicate" | "failed";

type ActivityItem = {
  id: string;
  timestamp: string;
  title: string;
  department: Department;
  recipients: number;
  priority: Priority;
  status: ActivityStatus;
  detail: string;
};

type DashboardMetrics = {
  totalSent: number;
  successRate: number;
  latestBatch: number;
  healthLabel: string;
};

const departments: Department[] = ["CSE", "ECE", "ME", "CIVIL", "EEE"];
const priorities: Priority[] = ["high", "normal", "low"];

function buildIdempotencyKey(): string {
  return `notif-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
}

function formatTime(date: Date): string {
  return new Intl.DateTimeFormat("en-IN", {
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
    hour12: false,
  }).format(date);
}

function getStatusClasses(status: ActivityStatus): string {
  switch (status) {
    case "sent":
      return "bg-emerald-50 text-emerald-700";
    case "duplicate":
      return "bg-amber-50 text-amber-700";
    case "failed":
      return "bg-rose-50 text-rose-700";
    default:
      return "bg-slate-100 text-slate-700";
  }
}

function getHealthLabel(health: HealthState): string {
  switch (health) {
    case "healthy":
      return "Cluster healthy";
    case "offline":
      return "Backend offline";
    case "degraded":
    default:
      return "Partially available";
  }
}

function toDepartment(value: string): Department {
  return departments.includes(value as Department)
    ? (value as Department)
    : "CSE";
}

function toPriority(value: string): Priority {
  return priorities.includes(value as Priority)
    ? (value as Priority)
    : "normal";
}

function buildActivityItemFromSuccess(
  result: NotificationResponse,
  title: string,
  priority: Priority,
  department: Department,
): ActivityItem {
  return {
    id: result.notification_id,
    timestamp: formatTime(new Date(result.created_at)),
    title: title.trim(),
    department,
    recipients: result.queued_deliveries,
    priority,
    status: result.duplicate ? "duplicate" : "sent",
    detail: result.status,
  };
}

function buildActivityItemFromRecentNotification(
  notification: RecentNotification,
): ActivityItem {
  return {
    id: notification.id,
    timestamp: formatTime(new Date(notification.created_at)),
    title: notification.title,
    department: toDepartment(notification.target_department),
    recipients: notification.queued_deliveries,
    priority: toPriority(notification.priority),
    status: notification.status === "failed" ? "failed" : "sent",
    detail: notification.status,
  };
}

function buildActivityItemFromError(
  title: string,
  department: Department,
  priority: Priority,
  message: string,
): ActivityItem {
  return {
    id: `failed-${Date.now()}`,
    timestamp: formatTime(new Date()),
    title: title.trim() || "Untitled",
    department,
    recipients: 0,
    priority,
    status: "failed",
    detail: message,
  };
}

export default function Dashboard(): JSX.Element {
  const [title, setTitle] = useState<string>("");
  const [message, setMessage] = useState<string>("");
  const [department, setDepartment] = useState<Department>("CSE");
  const [priority, setPriority] = useState<Priority>("normal");
  const [submitting, setSubmitting] = useState<boolean>(false);
  const [health, setHealth] = useState<HealthState>("degraded");
  const [flash, setFlash] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [activity, setActivity] = useState<ActivityItem[]>([]);

  useEffect((): (() => void) => {
    let mounted = true;

    const pollHealth = async (): Promise<void> => {
      try {
        const result = await getHealth();
        if (mounted) {
          setHealth(result.status === "ok" ? "healthy" : "degraded");
        }
      } catch {
        if (mounted) {
          setHealth("offline");
        }
      }
    };

    void pollHealth();
    const timerId = window.setInterval(() => {
      void pollHealth();
    }, 10000);

    return (): void => {
      mounted = false;
      window.clearInterval(timerId);
    };
  }, []);

  useEffect((): (() => void) => {
    let mounted = true;

    const loadRecentNotifications = async (): Promise<void> => {
      try {
        const notifications = await getRecentNotifications(5);
        if (!mounted) {
          return;
        }

        setActivity(
          notifications.map((notification) =>
            buildActivityItemFromRecentNotification(notification),
          ),
        );
      } catch {
        if (mounted) {
          setActivity([]);
        }
      }
    };

    void loadRecentNotifications();

    return (): void => {
      mounted = false;
    };
  }, []);

  const metrics = useMemo<DashboardMetrics>(() => {
    const totalSent = activity.reduce((sum, item) => sum + item.recipients, 0);
    const successfulCount = activity.filter(
      (item) => item.status !== "failed",
    ).length;
    const successRate =
      activity.length === 0
        ? 100
        : Math.round((successfulCount / activity.length) * 1000) / 10;

    return {
      totalSent,
      successRate,
      latestBatch: activity[0]?.recipients ?? 0,
      healthLabel: getHealthLabel(health),
    };
  }, [activity, health]);

  async function handleSubmit(
    event: FormEvent<HTMLFormElement>,
  ): Promise<void> {
    event.preventDefault();

    setSubmitting(true);
    setFlash(null);
    setError(null);

    try {
      const idempotencyKey = buildIdempotencyKey();
      const result = await sendNotification(
        {
          title,
          message,
          target_department: department,
          priority,
        },
        idempotencyKey,
      );

      const nextItem = buildActivityItemFromSuccess(
        result,
        title,
        priority,
        department,
      );
      setActivity((current) => [nextItem, ...current].slice(0, 5));
      setFlash(
        nextItem.status === "duplicate"
          ? `Existing idempotency key detected. ${department} reused the previous sent notification.`
          : `${department} notification batch sent successfully.`,
      );
      setTitle("");
      setMessage("");
    } catch (submissionError: unknown) {
      const submissionMessage =
        submissionError instanceof Error
          ? submissionError.message
          : "Unable to queue notification";

      setError(submissionMessage);
      setActivity((current) =>
        [
          buildActivityItemFromError(
            title,
            department,
            priority,
            submissionMessage,
          ),
          ...current,
        ].slice(0, 5),
      );
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <main className="min-h-screen px-3 py-4 sm:px-6 lg:px-10">
      <div className="mx-auto max-w-7xl">
        {/* <header className="panel mb-8 rounded-[30px] px-5 py-5 shadow-panel sm:px-6">
          <div className="flex flex-col gap-5 xl:flex-row xl:items-center xl:justify-between">
            <div className="flex items-center gap-4">
              <div className="flex h-14 w-14 items-center justify-center rounded-3xl bg-coral text-3xl font-black text-white shadow-soft">
                A
              </div>
              <div>
                <p className="font-display text-3xl uppercase tracking-tight text-ink">
                  AGH
                </p>
                <p className="text-sm text-slate-500">
                  Notification operations console
                </p>
              </div>
            </div>

            <nav className="flex flex-wrap items-center gap-4 text-sm font-semibold text-slate-400 sm:gap-6">
              <span className="border-b-2 border-coral pb-2 text-ink">
                Dashboard
              </span>
            </nav>

            <div className="flex items-center gap-3 sm:gap-4" />
          </div>
        </header> */}

        <section className="mb-10">
          <p className="mb-3 inline-flex rounded-full bg-white/80 px-4 py-2 text-xs font-semibold uppercase tracking-[0.28em] text-slate-500 shadow-soft">
            Department delivery control
          </p>
          <h1 className="font-display text-3xl leading-tight text-ink sm:text-5xl lg:text-6xl">
            System Overview
          </h1>
          <p className="mt-3 max-w-2xl text-base text-slate-500 sm:text-lg">
            Responsive broadcast console for queueing notification batches by
            department against your Go backend.
          </p>
        </section>

        <section className="data-grid  mb-8">
          <article className="panel rounded-4xl p-6 shadow-panel">
            <p className="mb-5 text-xs font-semibold uppercase tracking-[0.3em] text-slate-400">
              Total sent
            </p>
            <div className="text-4xl font-black tracking-tight text-ink sm:text-5xl">
              {metrics.totalSent.toLocaleString()}
            </div>
            <p className="mt-3 text-sm font-semibold text-emerald-600">
              Backend activity aggregate
            </p>
          </article>

          <article className="panel rounded-4xl p-6 shadow-panel">
            <p className="mb-5 text-xs font-semibold uppercase tracking-[0.3em] text-slate-400">
              Backend status
            </p>
            <div className="flex items-end gap-2">
              <span className="text-4xl font-black tracking-tight text-ink sm:text-5xl">
                {health === "healthy"
                  ? "OK"
                  : health === "offline"
                    ? "OFF"
                    : "WARN"}
              </span>
              <span className="pb-2 text-lg text-slate-300">/health</span>
            </div>
            <p className="mt-3 text-sm font-semibold text-slate-600">
              {metrics.healthLabel}
            </p>
          </article>

          <article className="panel rounded-4xl p-6 shadow-panel">
            <p className="mb-5 text-xs font-semibold uppercase tracking-[0.3em] text-slate-400">
              Last batch
            </p>
            <div className="flex items-end gap-2">
              <span className="text-4xl font-black tracking-tight text-ink sm:text-5xl">
                {metrics.latestBatch}
              </span>
              <span className="pb-2 text-lg text-slate-300">users</span>
            </div>
            <p className="mt-3 text-sm font-semibold text-slate-600">
              Most recent department send size
            </p>
          </article>

          <article className="panel rounded-4xl p-6 shadow-panel">
            <p className="mb-5 text-xs font-semibold uppercase tracking-[0.3em] text-slate-400">
              Success rate
            </p>
            <div className="flex items-center gap-4">
              <div className="grid h-20 w-20 place-items-center rounded-full border-[7px] border-coral text-2xl font-black text-ink">
                {metrics.successRate}%
              </div>
              <div>
                <div className="text-3xl font-black text-ink sm:text-4xl">
                  {metrics.successRate}%
                </div>
                <p className="text-sm font-semibold text-slate-400">
                  Backend-driven activity metric
                </p>
              </div>
            </div>
          </article>
        </section>

        <section className="grid gap-8 lg:grid-cols-[minmax(0,380px)_minmax(0,1fr)]">
          <article className="panel rounded-[36px] shadow-panel">
            <div className="border-b border-slate-100 px-6 py-6 sm:px-8">
              <div className="flex items-center gap-4">
                <div className="grid h-14 w-14 place-items-center rounded-full bg-rose-50 text-lg font-black text-coral shadow-soft">
                  TX
                </div>
                <div>
                  <h2 className="font-display text-3xl text-ink">Quick Send</h2>
                  <p className="text-sm text-slate-500">
                    Broadcast to an entire department
                  </p>
                </div>
              </div>
            </div>

            <form
              onSubmit={handleSubmit}
              className="space-y-6 px-6 py-8 sm:px-8"
            >
              <label className="block">
                <span className="mb-3 block text-xs font-semibold uppercase tracking-[0.3em] text-slate-400">
                  Notification title
                </span>
                <input
                  value={title}
                  onChange={(event) => setTitle(event.target.value)}
                  placeholder="e.g. Placement Cell Update"
                  className="w-full rounded-3xl border border-slate-200 bg-white px-5 py-4 text-base outline-none transition focus:border-coral"
                  required
                />
              </label>

              <label className="block">
                <span className="mb-3 block text-xs font-semibold uppercase tracking-[0.3em] text-slate-400">
                  Message content
                </span>
                <textarea
                  value={message}
                  onChange={(event) => setMessage(event.target.value)}
                  placeholder="Type your message here..."
                  rows={5}
                  className="w-full rounded-3xl border border-slate-200 bg-white px-5 py-4 text-base outline-none transition focus:border-coral"
                  required
                />
              </label>

              <div className="grid gap-4 sm:grid-cols-2">
                <label className="block">
                  <span className="mb-3 block text-xs font-semibold uppercase tracking-[0.3em] text-slate-400">
                    Department
                  </span>
                  <select
                    value={department}
                    onChange={(event) =>
                      setDepartment(toDepartment(event.target.value))
                    }
                    className="w-full rounded-3xl border border-slate-200 bg-white px-5 py-4 text-base outline-none transition focus:border-coral"
                  >
                    {departments.map((item) => (
                      <option key={item} value={item}>
                        {item}
                      </option>
                    ))}
                  </select>
                </label>

                <label className="block">
                  <span className="mb-3 block text-xs font-semibold uppercase tracking-[0.3em] text-slate-400">
                    Priority
                  </span>
                  <select
                    value={priority}
                    onChange={(event) =>
                      setPriority(event.target.value as Priority)
                    }
                    className="w-full rounded-3xl border border-slate-200 bg-white px-5 py-4 text-base outline-none transition focus:border-coral"
                  >
                    {priorities.map((item) => (
                      <option key={item} value={item}>
                        {item}
                      </option>
                    ))}
                  </select>
                </label>
              </div>

              <div className="rounded-3xl bg-slate-50 px-5 py-4 text-sm text-slate-500">
                Target department:{" "}
                <span className="font-bold text-ink">{department}</span>
                <span className="mt-2 block text-xs uppercase tracking-[0.25em] text-slate-400">
                  Backend resolves the matching users and reports successful
                  requests as sent
                </span>
              </div>

              {flash ? (
                <div className="rounded-3xl bg-emerald-50 px-5 py-4 text-sm font-semibold text-emerald-700">
                  {flash}
                </div>
              ) : null}

              {error ? (
                <div className="rounded-3xl bg-rose-50 px-5 py-4 text-sm font-semibold text-rose-700">
                  {error}
                </div>
              ) : null}

              <button
                type="submit"
                disabled={submitting}
                className="w-full rounded-3xl bg-coral px-6 py-4 text-lg font-black uppercase tracking-wide text-white shadow-soft transition hover:bg-[#ef301d] disabled:cursor-not-allowed disabled:opacity-60"
              >
                {submitting ? "Broadcasting..." : "Broadcast Now"}
              </button>
            </form>
          </article>

          <article className="panel rounded-[36px] shadow-panel">
            <div className="flex flex-col gap-3 border-b border-slate-100 px-6 py-6 sm:flex-row sm:items-center sm:justify-between sm:px-8">
              <div className="flex items-center gap-4">
                <div className="grid h-14 w-14 place-items-center rounded-full bg-slate-100 text-lg font-black text-slate-500">
                  LG
                </div>
                <div>
                  <h2 className="font-display text-3xl text-ink">
                    Last 5 Notifications
                  </h2>
                  <p className="text-sm text-slate-500">
                    Recent records fetched from the backend
                  </p>
                </div>
              </div>

              <Link
                href="/notifications"
                className="inline-flex items-center rounded-full bg-coral px-4 py-2 text-sm font-black uppercase tracking-wide text-white transition hover:bg-[#ef301d]"
              >
                View all
              </Link>
            </div>

            <div className="space-y-3 p-4 md:hidden">
              {activity.map((item) => (
                <article
                  key={item.id}
                  className="rounded-2xl border border-slate-100 bg-white/70 p-4"
                >
                  <div className="mb-2 flex items-center justify-between gap-3">
                    <p className="text-xs font-semibold uppercase tracking-wide text-slate-400">
                      {item.timestamp}
                    </p>
                    <span
                      className={`rounded-full px-3 py-1 text-[10px] font-bold uppercase tracking-wide ${getStatusClasses(item.status)}`}
                    >
                      {item.status}
                    </span>
                  </div>
                  <p className="text-base font-black text-ink break-words">{item.title}</p>
                  <p className="mt-1 text-sm text-slate-500">{item.detail}</p>
                  <div className="mt-3 flex items-center justify-between gap-3 text-xs font-semibold uppercase text-slate-500">
                    <span className="rounded-full bg-slate-100 px-3 py-1 text-slate-600">
                      {item.department}
                    </span>
                    <span>Recipients: {item.recipients}</span>
                  </div>
                </article>
              ))}
              {activity.length === 0 ? (
                <div className="rounded-2xl border border-slate-100 bg-white/70 p-6 text-center text-sm text-slate-400">
                  No notification activity available yet.
                </div>
              ) : null}
            </div>

            <div className="hidden overflow-x-auto md:block">
              <table className="min-w-[560px] w-full border-separate border-spacing-0 text-left">
                <thead>
                  <tr className="text-xs font-semibold uppercase tracking-[0.3em] text-slate-400">
                    <th className="px-4 py-4 sm:px-8 sm:py-5">Timestamp</th>
                    <th className="hidden px-4 py-4 sm:table-cell sm:px-8 sm:py-5">
                      Department
                    </th>
                    <th className="hidden px-4 py-4 md:table-cell sm:px-8 sm:py-5">
                      Recipients
                    </th>
                    <th className="px-6 py-5 sm:px-8">Type</th>
                    <th className="px-6 py-5 sm:px-8">Status</th>
                  </tr>
                </thead>
                <tbody>
                  {activity.map((item) => (
                    <tr
                      key={item.id}
                      className="border-t border-slate-100 text-base text-ink"
                    >
                      <td className="px-4 py-4 font-semibold text-slate-400 sm:px-8 sm:py-5">
                        {item.timestamp}
                      </td>
                      <td className="hidden px-4 py-4 sm:table-cell sm:px-8 sm:py-5">
                        <span className="rounded-full bg-slate-100 px-3 py-1 text-xs font-bold uppercase tracking-wide text-slate-600">
                          {item.department}
                        </span>
                      </td>
                      <td className="hidden px-4 py-4 font-semibold md:table-cell sm:px-8 sm:py-5">
                        {item.recipients}
                      </td>
                      <td className="px-6 py-5 sm:px-8">
                        <div className="font-black break-words">{item.title}</div>
                        <div className="mt-1 text-sm text-slate-400">
                          {item.detail}
                        </div>
                      </td>
                      <td className="px-6 py-5 sm:px-8">
                        <span
                          className={`rounded-full px-4 py-2 text-xs font-bold uppercase tracking-wide ${getStatusClasses(item.status)}`}
                        >
                          {item.status}
                        </span>
                      </td>
                    </tr>
                  ))}
                  {activity.length === 0 ? (
                    <tr>
                      <td
                        colSpan={5}
                        className="px-4 py-8 text-center text-sm text-slate-400 sm:px-8 sm:py-10"
                      >
                        No notification activity available yet.
                      </td>
                    </tr>
                  ) : null}
                </tbody>
              </table>
            </div>
          </article>
        </section>
      </div>
    </main>
  );
}
