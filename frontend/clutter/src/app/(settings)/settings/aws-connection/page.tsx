import IamRole from "./components/IamRole";

export default function AwsConnectionPage() {
  return (
    <div className="min-h-screen px-6 py-10">
      <div className="mx-auto max-w-7xl">
        <IamRole />
      </div>
    </div>
  );
}
