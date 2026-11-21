# Demo in ten minutes

How to show Clutter to someone in about ten minutes: sign in, onboarding, connections, one diagram deployed to two clouds at once, a live change, and destroy. Each step names what to click, what appears, and one sentence to say. The times come from a dry run on real AWS and VEXXHOST.

Pick one of two setups. **Local** costs nothing and shows sign-up and onboarding. **Real clouds** shows real AWS and VEXXHOST, with a team that is already linked.

## Before the demo (not part of the ten minutes)

| | Local | Real clouds |
| --- | --- | --- |
| Clouds | LocalStack and DevStack running (`npm run localstack -- start`, `npm run devstack -- start`, `npm run devstack -- route`) | a real AWS account linked and a VEXXHOST project connected, once (`backend/devserver/README.md` § Real clouds from this machine) |
| Clutter | `npm run stop -- --wipe` (an empty database, so onboarding shows), then `npm run dev -- --aws localstack` | `npm run dev`; its first line must say real AWS |
| Browser | one window at <http://localhost:3000>, signed out | signed in at `/dev/login`, on the linked team's Projects page |
| Check | `npm run doctor` says DevStack answers | Connections shows both clouds Verified; both clouds hold nothing of Clutter's |

On real clouds the run costs a few cents: the servers live for about five minutes.

## The ten minutes

| Time | Do | What appears | Say |
| --- | --- | --- | --- |
| 0:00 | Open <http://localhost:3000>. Move the pointer over the hero; scroll to **See how each cloud builds it** and switch Blocks, AWS, OpenStack. | The landing page; the same diagram as each cloud builds it. | "You draw once, in plain blocks. Clutter writes the Terraform for each cloud you link." |
| 1:00 | **Local:** sign in at `/dev/login`. Answer the four onboarding steps: experience, team name, first cloud (**Not sure yet**), **Full app**, **Open the editor**. **Real clouds:** open the team switcher, **New team**, to show the team step, then switch back to the linked team. | The editor opens on "My first project", with "Get to your first deploy". | "A new user is in the editor within a minute, with a working example." |
| 2:30 | **Local:** **Connect a cloud**, **Amazon Web Services**, **Connect local AWS**; then **OpenStack**, **Connect local OpenStack**. **Real clouds:** open **Connections** to show both Verified, then **New diagram**, **Full app**. | Both connections say "Clutter checked it and it works". | "Users link their own accounts. AWS through a role they create, OpenStack through an application credential. Clutter holds no passwords." |
| 3:30 | **View as** AWS, then OpenStack. | On OpenStack the bucket, function, table and API are marked "Not on OpenStack". | "Not every cloud builds everything, and Clutter says so before anything runs." |
| 4:00 | **Deploy** tab, **Add deployment**, the OpenStack connection. | It suggests running the four AWS-only blocks on AWS; the card reads "openstack + AWS". | "One deployment, two clouds: servers on OpenStack, the serverless part on AWS." |
| 4:30 | **Preview code**, then **Plan**. | The Terraform files per cloud; "Plan: … to add". | "You can read every line before it runs." |
| 5:00 | **Deploy**, then **Deploy** in the confirmation. Talk while it runs (about 90 seconds on real clouds). | The confirmation names the costly part (the NAT gateway); the card goes Applying, then Live; "Built 10 blocks (48 cloud resources)". | "Each run gets keys that last an hour at most, and only for this account." |
| 6:45 | In **What was built**, **Open** the website, then **Open** the API. | The web page (on OpenStack) and the API's answer (on AWS). | "Both clouds, one diagram, working together." |
| 7:15 | Select **web**, set **Size** to **medium**, **Save**, then **Update** (about 40 seconds). | "Changes not deployed yet", then "Built 10 blocks (1 changed)"; the page still opens after the server restarts. | "Changes are plans too: only what changed is touched." |
| 8:15 | **Destroy**, **Destroy for good** (about 30 seconds). | "Destroyed"; "What was built" is gone. | "One click removes everything the deployment built." |
| 9:00 | The project's **Deployments** tab; a card's **Show log**, **Copy**. | Every deployment with its state; the plain log. | "Everything is visible: what ran, where, and what it did." |

## If something goes wrong

| Problem | Do |
| --- | --- |
| A deploy takes long | Keep talking through steps 6:45 and 9:00 on another diagram; come back. |
| A real cloud refuses something | The card leads with the cloud's reason and a line of advice. Show it: it is part of the product. Then **Destroy**. |
| No internet, or a cloud is down | Switch to the local setup, or deploy to the **Mock cloud** (Connections, **Test cloud**, token `mock-valid-token`): it builds nothing, and every screen still works. |

## After the demo

Real clouds: check each cloud holds nothing of Clutter's (the run's Destroy does this; the clouds' own consoles confirm it). Local: `npm run stop` keeps the data; `npm run stop -- --wipe` clears it.
