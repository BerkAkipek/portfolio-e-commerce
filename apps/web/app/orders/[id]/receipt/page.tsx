import ReceiptClient from "./receipt-client";

export default async function ReceiptPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;

  return (
    <div className="relative min-h-screen overflow-hidden bg-[#070b14] text-slate-100 print:bg-white print:text-black">
      <div className="pointer-events-none absolute inset-0 print:hidden">
        <div className="absolute left-1/2 top-[-16rem] h-[36rem] w-[36rem] -translate-x-1/2 rounded-full bg-cyan-500/15 blur-3xl" />
        <div className="absolute bottom-[-12rem] right-[-8rem] h-[28rem] w-[28rem] rounded-full bg-emerald-500/10 blur-3xl" />
      </div>
      <ReceiptClient orderID={id} />
    </div>
  );
}
