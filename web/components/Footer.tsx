export default function Footer() {
  return (
    <footer className="py-10 border-t border-white/10 bg-black/30">
      <div className="container mx-auto max-w-6xl px-6 text-sm text-white/70 flex items-center justify-between">
        <span>© {new Date().getFullYear()} AURA</span>
        <div className="flex items-center gap-4">
          <a href="/docs/BRAND_GUIDE.md" className="hover:underline">Brand</a>
          <a href="/docs/UI_GUIDE.md" className="hover:underline">UI</a>
          <a href="https://github.com/Armour007/aura" target="_blank" rel="noreferrer" className="hover:underline">GitHub</a>
        </div>
      </div>
    </footer>
  )
}
