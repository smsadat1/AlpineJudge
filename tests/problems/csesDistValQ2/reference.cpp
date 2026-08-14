// Distinct Values Queries
// https://cses.fi/problemset/task/1734
//
// Solution 2: Fenwick array.
//
// (I stole this idea from the USACO tutorial.) We go through the array from
// left to right and maintain binary Fenwick array T so that at position j,
// T[i] = 1 iff. A[i] is the last occurence of the value A[i] in A[0..j).
//
// A = 3 1 4 1 5 9 2 6 5...
// T = 1 0 1 1 0 1 1 1 1
//
// That means the number of 1 bits in T is equal to the number of distinct
// elements in A[0..j) and the number of distinct elements in A[i..j) is
// equal to the Fenwick sum of T[i..j).
//
// The implementation below does the same except moving from right to left,
// so I can answer every query with a single prefix-sum.
//
// Time complexity: O(N + Q log N)

#include <bits/stdc++.h>

#include "fenwick.h"

using namespace std;

#define FOR(i,a,b) for (int i = int(a), _end_##i = int(b); i < _end_##i; ++i)
#define REP(i,n) FOR(i,0,n)

struct Query {
  int a, b, index;
};

int main() {
  // Make C++ I/O not slow. It's sad that this is necessary :-(
  ios_base::sync_with_stdio(false), cin.tie(nullptr);

  int N = 0, Q = 0;
  cin >> N >> Q;
  vector<int> A(N);
  for (auto &x : A) cin >> x;

  vector<Query> queries(Q);
  REP(i, Q) {
    int a = 0, b = 0;
    cin >> a >> b;
    --a;
    queries[i] = Query{.a = a, .b = b, .index=i};
  }

  // Sort by left endpoint, descending.
  sort(queries.begin(), queries.end(), [](const Query &q, const Query &r) {
    return q.a > r.a;
  });

  map<int, int> first_index;
  vector<int> answers(Q);
  vector<int> distinct(N);
  int i = N;
  for (auto [a, b, index] : queries) {
    while (i > a) {
      --i;
      int &last_i = first_index[A[i]];
      if (last_i > i) fenwick_update(distinct, last_i, -1);
      fenwick_update(distinct, last_i = i, 1);
    }
    answers[index] = fenwick_prefixsum(distinct, b);
  }

  for (int a : answers) cout << a << '\n';
}