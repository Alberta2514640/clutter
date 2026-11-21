import type { IsoVariant } from "./iso";
import IsoStack from "./IsoStack";

type Props = {
  // "outline" (a hairline, faint faces), "solid" (a block kind's colours) or
  // "brand" (the logo's colours). Default "outline".
  variant?: IsoVariant;
  // The block kind whose colours a solid block takes ("vm", "bucket").
  kind?: string;
  // Pixel width of the block (rounded to an even number). Default 24.
  size?: number;
  // The colour behind the block (IsoStack's paper).
  paper?: string;
  className?: string;
};

// IsoBlock is one block of the logo's grid. Decorative.
export default function IsoBlock({ variant = "outline", kind, size = 24, paper, className }: Props) {
  return <IsoStack blocks={[{ x: 0, y: 0, variant, kind }]} size={size} paper={paper} className={className} />;
}
