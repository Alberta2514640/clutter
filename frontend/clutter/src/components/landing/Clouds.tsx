import motion from "./motion.module.css";

// Clouds are named in plain text only, never with a logo (ADR-022).
const clouds = [
  {
    name: "Amazon Web Services",
    status: "Available",
    lines: [
      "Link your account with a CloudFormation stack that creates a role for Clutter. Clutter holds no AWS keys.",
      "Builds every block: VMs, networks, firewalls, volumes, buckets, functions, tables and HTTP APIs.",
      "Choose the region per connection, such as Canada (Central).",
    ],
  },
  {
    name: "OpenStack",
    status: "Available",
    lines: [
      "Any OpenStack cloud, public or private, including one you run yourself.",
      "Connect with an application credential. You can drop in the clouds.yaml file your cloud gives you.",
      "Builds VMs, networks, firewalls and volumes.",
    ],
  },
  {
    name: "Patrii Cloud",
    status: "Planned",
    lines: ["A Canadian cloud. Clutter plans to support it once Patrii opens its API to customers."],
  },
];

export function Clouds() {
  return (
    <section id="clouds" aria-labelledby="clouds-title" className="scroll-mt-16 border-t border-slate-800/60 bg-gradient-to-b from-black via-slate-950 to-black">
      <div className="mx-auto max-w-6xl px-4 py-16 md:px-6 md:py-24">
        <div className={`mb-10 max-w-2xl ${motion.reveal}`}>
          <p className="mb-3 text-sm font-semibold uppercase tracking-[0.2em] text-cyan-300">Clouds</p>
          <h2 id="clouds-title" className="text-3xl font-semibold tracking-tight text-white md:text-4xl">
            Clouds you can deploy to
          </h2>
          <p className="mt-4 leading-relaxed text-slate-300">One diagram can have a deployment on each cloud you link. More clouds are added one at a time.</p>
        </div>

        <ul className={`grid gap-6 md:grid-cols-3 ${motion.reveal}`}>
          {clouds.map((c) => (
            <li
              key={c.name}
              className="rounded-2xl border border-slate-800 bg-slate-950/70 p-6 transition-[transform,border-color] duration-300 hover:-translate-y-1 hover:border-slate-600 motion-reduce:transition-none motion-reduce:hover:translate-y-0">
              <div className="mb-4 flex items-center justify-between gap-3">
                <h3 className="text-lg font-semibold text-white">{c.name}</h3>
                <span
                  className={`rounded-full border px-2.5 py-0.5 text-xs font-medium ${
                    c.status === "Available" ? "border-teal-400/40 bg-teal-400/10 text-teal-200" : "border-slate-600 bg-slate-800/60 text-slate-300"
                  }`}>
                  {c.status}
                </span>
              </div>
              <ul className="space-y-2 text-sm leading-relaxed text-slate-300">
                {c.lines.map((l) => (
                  <li key={l}>{l}</li>
                ))}
              </ul>
            </li>
          ))}
        </ul>
        <p className="mt-6 text-sm text-slate-400">Cloud names belong to their owners. Clutter is not made or endorsed by them.</p>
      </div>
    </section>
  );
}
