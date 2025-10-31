"use client"
import Navbar from '@/components/Navbar'
import Footer from '@/components/Footer'

export default function Onboard() {
  return (
    <main className="bg-deep text-mist min-h-dvh">
      <Navbar />
      <div className="container mx-auto max-w-2xl px-6 py-20 space-y-6">
        <h1 className="text-3xl font-semibold">Welcome to AURA</h1>
        <p className="text-white/80">Set up your first organization, invite teammates, and create your first agent.</p>
        <div className="rounded-2xl border border-white/10 p-6 bg-white/5 space-y-4">
          <input className="w-full rounded-xl bg-black/30 border border-white/10 px-4 py-3" placeholder="Organization name" />
          <button className="px-4 py-3 rounded-2xl bg-plasma text-black font-medium">Create</button>
        </div>
      </div>
      <Footer />
    </main>
  )
}
