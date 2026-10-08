// A path id is a positive integer. A route that rejects its id doesn't match,
// so the URL falls through to the not-found route.
export const idParams = {
  parse: ({ id }: { id: string }) => {
    const parsed = Number(id);
    return /^[1-9]\d*$/.test(id) && Number.isSafeInteger(parsed) ? { id: parsed } : false;
  },
};
