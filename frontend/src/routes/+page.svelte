<script lang="ts">
  import Hero from '$lib/components/marketing/Hero.svelte';
  import FeatureGrid from '$lib/components/marketing/FeatureGrid.svelte';
  import HowItWorks from '$lib/components/marketing/HowItWorks.svelte';
  import CodeTabs from '$lib/components/marketing/CodeTabs.svelte';
  import LogosMarquee from '$lib/components/marketing/LogosMarquee.svelte';
  import CTASection from '$lib/components/marketing/CTASection.svelte';
  import TryIt from '$lib/components/marketing/TryIt.svelte';
  import Pricing from '$lib/components/marketing/Pricing.svelte';
  import Comparison from '$lib/components/marketing/Comparison.svelte';
  import DemoForm from '$lib/components/marketing/DemoForm.svelte';
</script>

<Hero />
<div id="features">
  <FeatureGrid />
</div>
<LogosMarquee />
<div id="how">
  <HowItWorks />
</div>
<CodeTabs />
<TryIt />
<Pricing />
<Comparison />
<CTASection />
<DemoForm />
