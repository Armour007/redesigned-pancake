"use client"
import Link from 'next/link'
import { Command, Github, BookOpen } from 'lucide-react'

// AURA UI REBUILD
export default function Navbar() {
  return (
    <nav className="sticky top-0 z-50 backdrop-blur-md bg-black/30 border-b border-white/10">
      <div className="container mx-auto max-w-7xl px-6 h-14 flex items-center justify-between">
        <Link href="/" className="flex items-center gap-2">
          <span className="inline-block w-6 h-6 rounded-xl bg-auraGradient breathing-glow" />
          <span className="font-semibold tracking-wide">AURA</span>
        </Link>
        <div className="flex items-center gap-4 text-sm">
          <a className="hover:underline text-white/80" href="/docs/BRAND_GUIDE.md">Brand</a>
          <a className="hover:underline text-white/80" href="/docs/UI_GUIDE.md">UI</a>
          <a className="hover:underline text-white/80" href="https://github.com/Armour007/aura" target="_blank" rel="noreferrer">GitHub</a>
        </div>
      </div>
    </nav>
  )
}
