import { NextRequest } from "next/server";

import { proxyJSONPost } from "../_utils";

export async function POST(request: NextRequest) {
  return proxyJSONPost(request, "/auth/login");
}
