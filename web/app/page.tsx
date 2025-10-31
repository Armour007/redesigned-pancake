"use client"
import { motion } from 'framer-motion'
import Link from 'next/link'
import dynamic from 'next/dynamic'
import Navbar from '@/components/Navbar'
import Hero from '@/components/Hero'
import Footer from '@/components/Footer'

const Hero3D = dynamic(() => import('@/components/Hero3D'), { ssr: false })

// AURA UI REBUILD
export default function Page() {
  return (
    <main className="bg-deep text-mist">
      <Navbar />
      <section className="relative overflow-hidden">
        <Hero />
      </section>
      <section className="relative">
        <Hero3D />
      </section>
      <section className="container mx-auto max-w-6xl px-6 py-16">
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
          {[
            { title: 'Trust Layer', desc: 'Policy, verification, and cryptographic identity.' },
            { title: 'Agent Builder', desc: 'Blocks, skills, and workflows for autonomous systems.' },
            { title: 'Team Collaboration', desc: 'Spaces, roles, and shared evaluations.' },
            { title: 'Developer API Hub', desc: 'OpenAPI, SDKs, and webhooks with signatures.' },
          ].map((f) => (
            <motion.div key={f.title} whileHover={{ y: -4 }} className="rounded-2xl border border-white/10 p-6 bg-white/5 backdrop-blur-md">
              <h3 className="text-lg font-semibold mb-2">{f.title}</h3>
              <p className="text-sm text-white/70">{f.desc}</p>
            </motion.div>
          ))}
        </div>
        <div className="mt-10 flex gap-4">
          <Link className="px-5 py-3 rounded-2xl bg-plasma text-black font-medium" href="/dashboard">Launch Workspace</Link>
          <a className="px-5 py-3 rounded-2xl border border-white/20" href="/docs/BRAND_GUIDE.md">Read Docs</a>
        </div>
      </section>
      <Footer />
    </main>
  )
}
