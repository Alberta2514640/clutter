export const allowed: Record<string, Record<string, string[]>> = {
  Lambda: { "reads-from": ["DynamoDB"], invokes: ["Lambda", "APIGateway"] },
  APIGateway: { invokes: ["Lambda"] },
};
export const isAllowed = (fromType: string, relation: string, toType: string) => !!allowed[fromType]?.[relation]?.includes(toType);
