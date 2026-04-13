import { SharedArray } from 'k6/data';

export const tokens = new SharedArray('tokens', function () {
  return JSON.parse(open('./tokens.json'));
});

export function getToken(vu) {
  if (!tokens.length) {
    throw new Error('tokens.json is empty; generate tokens before running load tests');
  }
  return tokens[vu % tokens.length];
}