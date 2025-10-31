"use client"
import { motion } from 'framer-motion'
import Navbar from '@/components/Navbar'
import Footer from '@/components/Footer'
import AgentCard from '@/components/AgentCard'

// AURA UI REBUILD
export default function Dashboard() {
  return (
    <main className="bg-deep text-mist min-h-dvh">
      <Navbar />
      <div className="container mx-auto max-w-7xl px-6 py-10 grid grid-cols-1 lg:grid-cols-5 gap-8">
        <aside className="lg:col-span-1 space-y-3">
          <div className="rounded-2xl border border-white/10 p-4 bg-white/5">Home</div>
          <div className="rounded-2xl border border-white/10 p-4 bg-white/5">Agents</div>
          <div className="rounded-2xl border border-white/10 p-4 bg-white/5">Teams</div>
          <div className="rounded-2xl border border-white/10 p-4 bg-white/5">Analytics</div>
        </aside>
        <section className="lg:col-span-4 space-y-6">
          <div className="rounded-2xl border border-white/10 p-6 bg-white/5">
            <h2 className="text-xl font-semibold mb-4">Chat Console</h2>
            <div className="h-40 rounded-xl bg-black/40 border border-white/10" />
          </div>
          <div className="grid md:grid-cols-3 gap-6">
            <AgentCard name="Atlas" status="Active" />
            <AgentCard name="Lyra" status="Idle" />
            <AgentCard name="Sol" status="Training" />
          </div>
          <div className="rounded-2xl border border-white/10 p-6 bg-white/5 flex items-center justify-between">
            <div>Credits: 82% used</div>
            <button className="px-4 py-2 rounded-2xl bg-plasma text-black font-medium">Upgrade</button>
          </div>
        </section>
      </div>
      <Footer />
    </main>
  )
}
