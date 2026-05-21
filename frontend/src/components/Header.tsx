export default function Header() {
  return (
    <div className="flex items-center gap-3 px-5 py-4 border-b border-[var(--color-border)]">
      <span className="text-2xl">🎙️</span>
      <div>
        <h1 className="text-lg font-semibold text-[var(--color-text-primary)] leading-none">
          Hanhan Radio
        </h1>
        <p className="text-xs text-[var(--color-text-secondary)] mt-1">
          AI 电台 DJ
        </p>
      </div>
    </div>
  );
}
