import TeamSettingsFrame from "./_components/TeamSettingsFrame";

// A server layout around the client frame, so each tab's page or layout can
// set its own browser-tab title. It sets none itself: a title here would stop
// the root "%s | Clutter" template reaching the tabs below it.
export default function TeamSettingsLayout({ children }: { children: React.ReactNode }) {
  return <TeamSettingsFrame>{children}</TeamSettingsFrame>;
}
